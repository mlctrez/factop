package service

import (
	"errors"
	"github.com/mlctrez/bind"
	"github.com/mlctrez/rcon"
	"github.com/nats-io/nats.go"
	"log/slog"
	"regexp"
	"strings"
	"sync"
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

func (r *Rcon) handle(msg *nats.Msg) {

	r.connMutex.Lock()
	defer r.connMutex.Unlock()
	if r.conn == nil {
		r.Nats.Reply(msg, nil, errors.New("rcon not connected"))
		return
	}
	result, err := r.conn.Execute(string(msg.Data))
	if err != nil {
		r.Nats.Reply(msg, nil, err)
		return
	}

	var newResult string
	parts := strings.Split(result, "\n")
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		// suppress [JOIN],[CHAT] and other console messages
		// r.Infof("command %q result %q", req.Payload, parts[i])
		if IgnorePattern.MatchString(parts[i]) {
			r.Info("rcon ignoring", "line", parts[i])
			continue
		}
		newResult = parts[i]
	}
	r.Nats.Reply(msg, []byte(newResult), nil)
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
