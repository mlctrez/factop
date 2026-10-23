package service

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"

	"github.com/mlctrez/bind"
	"github.com/mlctrez/rcon"
	"github.com/nats-io/nats.go"
)

var _ bind.Startup = (*Rcon)(nil)
var _ bind.Shutdown = (*Rcon)(nil)

type Rcon struct {
	slog.Logger
	Nats      *Nats
	Factorio  *Factorio
	conn      *rcon.Conn
	connMutex *sync.Mutex
}

func (r *Rcon) Startup() error {
	r.connMutex = &sync.Mutex{}
	if err := r.Nats.Subscribe("factorio.stdout", r.stdoutMonitor); err != nil {
		return err
	}
	if err := r.Nats.Subscribe(ConsoleSubject, r.handleConsole); err != nil {
		return err
	}
	return r.Nats.Subscribe("factop.rcon", r.handle)
}

func (r *Rcon) Shutdown() error {
	r.connMutex.Lock()
	defer r.connMutex.Unlock()
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

var IgnorePattern = regexp.MustCompile(`(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}) \[(\w+)] (.+)`)

// ConsoleSubject receives plain Factorio console commands (e.g. "/server-save")
// that are forwarded to RCON verbatim. Unlike factop.rcon, every non-console
// output line is returned, joined with newlines. It is meant for manual admin
// and maintenance commands; plugins and scripts that need game data should use
// factop.rcon with "/sc" Lua and return JSON via helpers.table_to_json.
const ConsoleSubject = "factop.rcon.console"

// handle executes msg.Data via RCON verbatim and replies with the last output
// line. Callers (focmd rcon, plugins) send their own "/sc ..." wrapping.
func (r *Rcon) handle(msg *nats.Msg) {
	result, err := r.execute(string(msg.Data))
	if err != nil {
		r.Nats.Reply(msg, nil, err)
		return
	}
	var newResult string
	if lines := r.filterOutput(result); len(lines) > 0 {
		newResult = lines[len(lines)-1]
	}
	r.Nats.Reply(msg, []byte(newResult), nil)
}

// handleConsole executes a single plain console command and replies with all
// output lines.
func (r *Rcon) handleConsole(msg *nats.Msg) {
	command, err := ConsoleCommand(string(msg.Data))
	if err != nil {
		r.Nats.Reply(msg, nil, err)
		return
	}
	r.Info("rcon console command", "command", command)
	result, err := r.execute(command)
	if err != nil {
		r.Nats.Reply(msg, nil, err)
		return
	}
	r.Nats.Reply(msg, []byte(strings.Join(r.filterOutput(result), "\n")), nil)
}

// ConsoleCommand validates and normalizes a plain console command. It must be
// a single line starting with "/" so that plain text is never broadcast to
// players as a chat message.
func ConsoleCommand(raw string) (string, error) {
	command := strings.TrimSpace(raw)
	switch {
	case command == "":
		return "", errors.New("console command is empty")
	case strings.ContainsAny(command, "\r\n"):
		return "", errors.New("console command must be a single line")
	case !strings.HasPrefix(command, "/"):
		return "", fmt.Errorf("console command must start with / (got %q); plain text would be sent as chat", command)
	}
	return command, nil
}

// execute sends command over the RCON connection.
func (r *Rcon) execute(command string) (string, error) {
	r.connMutex.Lock()
	defer r.connMutex.Unlock()
	if r.conn == nil {
		return "", errors.New("rcon not connected")
	}
	return r.conn.Execute(command)
}

// filterOutput splits an RCON response into lines, dropping blank lines and
// timestamped console messages such as [JOIN] and [CHAT].
func (r *Rcon) filterOutput(result string) []string {
	var lines []string
	for _, line := range strings.Split(result, "\n") {
		if line == "" {
			continue
		}
		if IgnorePattern.MatchString(line) {
			r.Info("rcon ignoring", "line", line)
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

const RconStartupMarker = "Starting RCON interface at IP ADDR"

func (r *Rcon) stdoutMonitor(msg *nats.Msg) {
	if strings.Contains(string(msg.Data), RconStartupMarker) {
		if IgnorePattern.MatchString(string(msg.Data)) {
			// in case a user types a console message containing RconStartupMarker
			r.Info("rcon ignoring", "line", string(msg.Data))
			return
		}

		r.Info("rcon monitor detected rcon startup")
		r.connMutex.Lock()
		defer r.connMutex.Unlock()

		var err error
		if r.conn != nil {
			if err = r.conn.Close(); err != nil {
				r.Error("error closing rcon connection (ignored)", "error", err)
			}
			r.conn = nil
		}

		// use a larger payload length for rcon commands
		opOne := rcon.SetMaxCommandLen(16384)
		// use a nonzero command id to skip getting console messages in rcon output
		opTwo := rcon.SetExeCommandId(10)

		r.conn, err = rcon.Dial(r.Factorio.RconBind, r.Factorio.RconPassword, opOne, opTwo)
		if err != nil {
			r.Error("error dialing rcon", "error", err)
		}

		// these will suppress the disable achievements warning when loading a brand
		// new save file so they don't show up in the user's console messaging or cause issues
		// with other rcon commands not executing
		_, _ = r.conn.Execute("/sc rcon.print(\"disabling achievements\")")
		_, _ = r.conn.Execute("/sc rcon.print(\"disabling achievements\")")
	}
}
