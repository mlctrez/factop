package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
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
	fmt.Println("  rcon <path>       - Send RCON command from file")
	fmt.Println("  lrcon <path>      - Send RCON command to local RCON via NATS")
}

func Watch() (err error) {
	var conn *nats.Conn
	if conn, err = nats.Connect(fmt.Sprintf("nats://%s", Host),
		nats.MaxReconnects(-1),
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

	fmt.Println("watching factorio.* and udp.* ...")
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

	_, err = conn.Subscribe("udp.*", func(msg *nats.Msg) {
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

func Rcon() (err error) {
	if len(os.Args) < 3 {
		return errors.New("usage: rcon <path>")
	}
	return runRcon(os.Args[2])
}

func runRcon(path string) (err error) {
	var conn *nats.Conn
	if conn, err = nats.Connect(fmt.Sprintf("nats://%s", Host)); err != nil {
		return err
	}
	defer conn.Close()

	var file []byte
	if file, err = os.ReadFile(path); err != nil {
		return err
	}
	scanner := bufio.NewScanner(bytes.NewReader(file))
	var lines []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "--") {
			lines = append(lines, line)
		}
	}

	return sendMessage(conn, "factop.rcon", []byte("/sc "+strings.Join(lines, "\n")))
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
