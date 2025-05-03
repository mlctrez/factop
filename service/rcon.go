package service

import (
	"context"
	"errors"
	"github.com/mlctrez/factop/api"
	"github.com/mlctrez/rcon"
	"github.com/mlctrez/servicego"
	"github.com/nats-io/nats.go"
	"regexp"
	"strings"
	"sync"
)

var _ Component = (*Rcon)(nil)

type Rcon struct {
	servicego.DefaultLogger
	Factorio  *Factorio
	context   context.Context
	conn      *rcon.Conn
	connMutex *sync.Mutex
}

func (r *Rcon) Start(s *Service) error {
	r.Logger(s.Log())
	r.context = s.context
	r.Factorio = s.Factorio
	r.connMutex = &sync.Mutex{}
	if err := s.Nats.Subscribe("factorio.stdout", r.stdoutMonitor); err != nil {
		return err
	}
	impl := api.NewRconHandler(s.context, s.Nats.conn, r)
	return s.Nats.Subscribe(impl.Subject(), impl.Handler)
}

func (r *Rcon) Stop() error {
	r.connMutex.Lock()
	defer r.connMutex.Unlock()
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

var IgnorePattern = regexp.MustCompile(`(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}) \[(\w+)] (.+)`)

func (r *Rcon) Execute(_ context.Context, req *api.RconCommand) (*api.RconResult, error) {
	r.connMutex.Lock()
	defer r.connMutex.Unlock()
	if r.conn == nil {
		return nil, errors.New("rcon not connected")
	}
	result, err := r.conn.Execute(req.Payload)
	if err != nil {
		return nil, err
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
			r.Infof("rcon ignoring %q", parts[i])
			continue
		}
		newResult = parts[i]
	}
	return &api.RconResult{Payload: newResult}, nil
}

const RconStartupMarker = "Starting RCON interface at IP ADDR"

func (r *Rcon) stdoutMonitor(msg *nats.Msg) {
	if strings.Contains(string(msg.Data), RconStartupMarker) {
		if IgnorePattern.MatchString(string(msg.Data)) {
			// in case a user types a console message containing RconStartupMarker
			r.Infof("rcon monitor ignoring %q", string(msg.Data))
			return
		}

		r.Infof("rcon monitor detected rcon startup")
		r.connMutex.Lock()
		defer r.connMutex.Unlock()

		var err error
		if r.conn != nil {
			if err = r.conn.Close(); err != nil {
				r.Errorf("error closing rcon connection (ignored) : %s", err)
			}
			r.conn = nil
		}

		// use a larger payload length for rcon commands
		opOne := rcon.SetMaxCommandLen(16384)
		// use a nonzero command id to skip getting console messages in rcon output
		opTwo := rcon.SetExeCommandId(10)

		r.conn, err = rcon.Dial(r.Factorio.RconBind, r.Factorio.RconPassword, opOne, opTwo)
		if err != nil {
			r.Errorf("error dialing rcon : %s", err)
		}

		// these will suppress the disable achievements warning when loading a brand
		// new save file so they don't show up in the user's console messaging or cause issues
		// with other rcon commands not executing
		_, _ = r.conn.Execute("/sc rcon.print(\"disabling achievements\")")
		_, _ = r.conn.Execute("/sc rcon.print(\"disabling achievements\")")
	}
}
