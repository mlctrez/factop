package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mlctrez/factop/softmod"
	"github.com/nats-io/nats.go"
)

var Host string

func main() {
	Host = os.Getenv("FACTOP_HOST")
	if Host == "" {
		Host = "factorio"
	}
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]
	var err error

	switch command {
	case "watch":
		err = Watch()
	case "service":
		err = Service()
	case "softmod":
		err = Softmod()
	case "command":
		err = Command()
	case "prcon":
		err = PRcon()
	case "rcon":
		err = Rcon()
	case "lrcon":
		err = LRcon()
	case "console":
		err = Console()
	case "plugin":
		err = Plugin()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		return
	}

	if err != nil {
		log.Fatal(err)
	}
}

func printUsage() {
	fmt.Println("Usage: go run cmd/main.go <command> [args...]")
	fmt.Println("Available commands:")
	fmt.Println("  watch             - Watch factorio.* and udp.* NATS messages")
	fmt.Println("  service           - Build and deploy the factop service")
	fmt.Println("  softmod           - Deploy the softmod zip")
	fmt.Println("  command <args...> - Send a command to factop.command")
	fmt.Println("  prcon <delay> <p> - Periodically send RCON command from file")
	fmt.Println("  rcon <path>       - Send Lua from file via RCON (/sc)")
	fmt.Println("  rcon -            - Send Lua read from stdin (also used when stdin is piped and no path is given)")
	fmt.Println("  rcon -e <lua...>  - Send inline Lua; remaining args are joined with spaces")
	fmt.Println("  lrcon <path>      - Send RCON command to local RCON via NATS")
	fmt.Println("  console <cmd...>  - Send a plain console command (e.g. /players) via RCON, no Lua wrapping")
	fmt.Println("  plugin <sub> ...  - Manage plugins (register, unregister, start, stop, restart, status, deploy, rollback, versions, version-remove, list)")
}

const WatchReconnectWait = 250 * time.Millisecond

func Watch() (err error) {
	var conn *nats.Conn
	if conn, err = nats.Connect(fmt.Sprintf("nats://%s", Host),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(WatchReconnectWait),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				fmt.Printf("Disconnected: %v\n", err)
			} else {
				fmt.Println("Disconnected")
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			fmt.Printf("Reconnected to %s\n", nc.ConnectedUrl())
		}),
	); err != nil {
		return err
	}
	defer conn.Close()

	fmt.Println("watching factorio.*, udp.>, factop.log, plugin.* ...")
	_, err = conn.Subscribe("factorio.*", func(msg *nats.Msg) {
		data := string(msg.Data)
		if msg.Subject == "factorio.softmod" {
			data = fmt.Sprintf("(binary data %d bytes)", len(msg.Data))
		}
		fmt.Printf("[%s] %s\n", msg.Subject, data)
	})
	if err != nil {
		return err
	}

	_, err = conn.Subscribe("udp.>", func(msg *nats.Msg) {
		if msg.Subject != "udp.incoming.game_tick" {
			fmt.Printf("[%s] %s\n", msg.Subject, string(msg.Data))
		}
	})
	if err != nil {
		return err
	}

	_, err = conn.Subscribe("factop.log", func(msg *nats.Msg) {
		fmt.Printf("[%s] %s\n", msg.Subject, string(msg.Data))
	})
	if err != nil {
		return err
	}

	_, err = conn.Subscribe("plugin.*", func(msg *nats.Msg) {
		fmt.Printf("[%s] %s\n", msg.Subject, string(msg.Data))
	})
	if err != nil {
		return err
	}

	// wait until interrupted
	select {}
}

func Service() (err error) {
	var tempDir string
	if tempDir, err = os.MkdirTemp("", "factop"); err != nil {
		return err
	} else {
		defer func() { _ = os.RemoveAll(tempDir) }()
	}

	binPath := filepath.Join(tempDir, "factop")
	// Replaced sh.Run with os/exec
	cmd := exec.Command("go", "build", "-o", binPath, "factop.go")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		return err
	}

	fmt.Println("deploying", filepath.Base(binPath), "to", Host)
	var binFile []byte
	if binFile, err = os.ReadFile(binPath); err != nil {
		return err
	}

	var req *http.Request
	u := fmt.Sprintf("https://%s:2000/deploy/%s", Host, "factop")
	if req, err = http.NewRequest("POST", u, bytes.NewBuffer(binFile)); err != nil {
		return err
	}

	var res *http.Response
	if res, err = http.DefaultClient.Do(req); err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return fmt.Errorf("incorrect status code %d", res.StatusCode)
	}
	fmt.Println("factop deploy success")
	return nil
}

func Softmod() (err error) {
	var conn *nats.Conn
	if conn, err = nats.Connect(fmt.Sprintf("nats://%s", Host)); err != nil {
		return err
	}
	defer conn.Close()

	var buffer *bytes.Buffer
	if buffer, err = softmod.CreateZip("save"); err != nil {
		return err
	}

	return sendMessage(conn, "factorio.softmod", buffer.Bytes())
}

func Command() (err error) {
	if len(os.Args) < 3 {
		return errors.New("usage: command <command> [args...]")
	}
	var conn *nats.Conn
	if conn, err = nats.Connect(fmt.Sprintf("nats://%s", Host)); err != nil {
		return err
	}
	defer conn.Close()
	return sendMessage(conn, "factop.command", []byte(strings.Join(os.Args[2:], " ")))
}

func PRcon() (err error) {
	if len(os.Args) < 4 {
		return errors.New("usage: prcon <delay_seconds> <path>")
	}
	delay, err := strconv.Atoi(os.Args[2])
	if err != nil {
		return fmt.Errorf("invalid delay: %v", err)
	}
	path := os.Args[3]

	ticker := time.NewTicker(time.Duration(delay) * time.Second)
	defer ticker.Stop()

	// Initial call
	if err = runRcon(path); err != nil {
		return err
	}

	for range ticker.C {
		if err = runRcon(path); err != nil {
			return err
		}
	}
	return nil
}

const rconUsage = "usage: rcon <path> | rcon - | rcon -e <lua...> (or pipe Lua to stdin with no args)"

func Rcon() (err error) {
	var src []byte
	if src, err = resolveRconSource(os.Args[2:], os.Stdin, stdinPiped()); err != nil {
		return err
	}
	return sendRcon(src)
}

// stdinPiped reports whether stdin is a pipe or file rather than a terminal.
func stdinPiped() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice == 0
}

// resolveRconSource determines where the Lua for "focmd rcon" comes from:
//
//	rcon <path>       read the file at path (original behavior)
//	rcon -            read stdin
//	rcon -e <lua...>  inline Lua, remaining args joined with spaces
//	rcon              read stdin when it is piped, otherwise a usage error
func resolveRconSource(args []string, stdin io.Reader, piped bool) ([]byte, error) {
	switch {
	case len(args) == 0:
		if !piped {
			return nil, errors.New(rconUsage)
		}
		return readRconStdin(stdin)
	case args[0] == "-e":
		if len(args) < 2 {
			return nil, errors.New("rcon -e requires inline Lua, e.g. rcon -e 'rcon.print(game.tick)'")
		}
		inline := strings.Join(args[1:], " ")
		if strings.TrimSpace(inline) == "" {
			return nil, errors.New("rcon -e: inline Lua is empty")
		}
		return []byte(inline), nil
	case len(args) > 1:
		return nil, errors.New(rconUsage)
	case args[0] == "-":
		return readRconStdin(stdin)
	default:
		return os.ReadFile(args[0])
	}
}

func readRconStdin(stdin io.Reader) ([]byte, error) {
	src, err := io.ReadAll(stdin)
	if err != nil {
		return nil, fmt.Errorf("reading stdin: %w", err)
	}
	if len(bytes.TrimSpace(src)) == 0 {
		return nil, errors.New("rcon: no Lua received on stdin")
	}
	return src, nil
}

// rconPayload converts Lua source into the /sc command sent to factop.rcon.
// Blank lines and lines starting with "--" are dropped and the remaining
// lines are trimmed.
func rconPayload(src []byte) []byte {
	scanner := bufio.NewScanner(bytes.NewReader(src))
	var lines []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "--") {
			lines = append(lines, line)
		}
	}
	return []byte("/sc " + strings.Join(lines, "\n"))
}

func runRcon(path string) (err error) {
	var src []byte
	if src, err = os.ReadFile(path); err != nil {
		return err
	}
	return sendRcon(src)
}

func sendRcon(src []byte) (err error) {
	var conn *nats.Conn
	if conn, err = nats.Connect(fmt.Sprintf("nats://%s", Host)); err != nil {
		return err
	}
	defer conn.Close()
	return sendMessage(conn, "factop.rcon", rconPayload(src))
}

// consoleSubject must match service.ConsoleSubject.
const consoleSubject = "factop.rcon.console"

// Console sends a plain Factorio console command such as "/players" through
// RCON without the /sc Lua wrapping used by the rcon command.
func Console() (err error) {
	var command string
	if command, err = consoleCommand(os.Args[2:]); err != nil {
		return err
	}
	var conn *nats.Conn
	if conn, err = nats.Connect(fmt.Sprintf("nats://%s", Host)); err != nil {
		return err
	}
	defer conn.Close()

	err = sendMessage(conn, consoleSubject, []byte(command))
	if errors.Is(err, nats.ErrNoResponders) {
		// Older factop without the console subject: factop.rcon also forwards
		// verbatim but only returns the last output line.
		fmt.Fprintln(os.Stderr, "warning: factop has no", consoleSubject, "handler; falling back to factop.rcon (only the last output line is returned)")
		err = sendMessage(conn, "factop.rcon", []byte(command))
	}
	return err
}

// consoleCommand joins the console arguments into a single command line.
// The command must start with "/" so plain text is never sent as chat.
func consoleCommand(args []string) (string, error) {
	command := strings.TrimSpace(strings.Join(args, " "))
	if command == "" {
		return "", errors.New("usage: console <command...>, e.g. console /players")
	}
	if strings.ContainsAny(command, "\r\n") {
		return "", errors.New("console command must be a single line")
	}
	if !strings.HasPrefix(command, "/") {
		return "", fmt.Errorf("console command must start with / (got %q); plain text would be sent as chat", command)
	}
	return command, nil
}

func LRcon() (err error) {
	if len(os.Args) < 3 {
		return errors.New("usage: lrcon <path>")
	}
	path := os.Args[2]
	var conn *nats.Conn
	if conn, err = nats.Connect("nats://localhost"); err != nil {
		return err
	}
	defer conn.Close()

	var file []byte
	if file, err = os.ReadFile(path); err != nil {
		return err
	}

	msg, err := conn.Request("rcon", file, 5*time.Second)
	if err != nil {
		return err
	}
	response := string(msg.Data)
	if response != "" {
		fmt.Println(response)
	}
	return nil
}

func Plugin() (err error) {
	if len(os.Args) < 3 {
		return errors.New("usage: plugin <subcommand> [args...]")
	}
	var conn *nats.Conn
	if conn, err = nats.Connect(fmt.Sprintf("nats://%s", Host)); err != nil {
		return err
	}
	defer conn.Close()

	subcmd := os.Args[2]

	switch subcmd {
	case "deploy":
		return pluginDeploy(conn)
	default:
		// All other commands: join subcmd + remaining args, send to factop.plugin
		payload := subcmd
		if len(os.Args) > 3 {
			payload = subcmd + " " + strings.Join(os.Args[3:], " ")
		}
		return sendMessage(conn, "factop.plugin", []byte(payload))
	}
}

func pluginDeploy(conn *nats.Conn) error {
	// Usage: plugin deploy <name> <version> <binary-path>
	if len(os.Args) < 6 {
		return errors.New("usage: plugin deploy <name> <version> <binary-path>")
	}
	name, version, binPath := os.Args[3], os.Args[4], os.Args[5]
	binData, err := os.ReadFile(binPath)
	if err != nil {
		return fmt.Errorf("reading binary: %w", err)
	}

	// 768KB chunk size — stays well under the default 1MB NATS max payload
	// after accounting for headers and subject overhead.
	const chunkSize = 768 * 1024

	if len(binData) <= chunkSize {
		// Small binary — send as single message with command in header.
		msg := nats.NewMsg("factop.plugin")
		msg.Header.Set("command", fmt.Sprintf("deploy %s %s", name, version))
		msg.Data = binData

		response, err := conn.RequestMsg(msg, 30*time.Second)
		if err != nil {
			return err
		}
		if errVal := response.Header.Get("error"); errVal != "" {
			return errors.New(errVal)
		}
		if len(response.Data) > 0 {
			fmt.Println(string(response.Data))
		}
		return nil
	}

	// Large binary — send as sequential chunks on factop.plugin.deploy.<name>.<version>.
	subject := fmt.Sprintf("factop.plugin.deploy.%s.%s", name, version)
	totalChunks := (len(binData) + chunkSize - 1) / chunkSize
	fmt.Printf("deploying %s v%s (%d bytes, %d chunks)\n", name, version, len(binData), totalChunks)

	for i := 0; i < totalChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(binData) {
			end = len(binData)
		}

		msg := nats.NewMsg(subject)
		msg.Header.Set("chunk-index", strconv.Itoa(i))
		msg.Header.Set("chunk-total", strconv.Itoa(totalChunks))
		msg.Data = binData[start:end]

		response, err := conn.RequestMsg(msg, 30*time.Second)
		if err != nil {
			return fmt.Errorf("sending chunk %d/%d: %w", i+1, totalChunks, err)
		}
		if errVal := response.Header.Get("error"); errVal != "" {
			return fmt.Errorf("chunk %d/%d: %s", i+1, totalChunks, errVal)
		}

		// Last chunk gets the final deploy response.
		if i == totalChunks-1 {
			if len(response.Data) > 0 {
				fmt.Println(string(response.Data))
			}
		} else {
			fmt.Printf("  chunk %d/%d sent\n", i+1, totalChunks)
		}
	}
	return nil
}

func sendMessage(conn *nats.Conn, subject string, data []byte) (err error) {
	response, err := conn.Request(subject, data, time.Second*10)
	if err != nil {
		return err
	}
	responseError := response.Header.Get("error")
	if responseError != "" {
		return errors.New(responseError)
	}
	if len(response.Data) > 0 {
		fmt.Println(string(response.Data))
	}
	return nil
}
