package service

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/mlctrez/bind"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

var _ bind.Startup = (*Nats)(nil)
var _ bind.Shutdown = (*Nats)(nil)

type Nats struct {
	slog.Logger
	LogHandler *LogHandler
	server     *server.Server
	conn       *nats.Conn
	subs       []*nats.Subscription
}

func (n *Nats) Startup() error {
	if err := n.newServer(); err != nil {
		return err
	}

	go n.server.Start()

	if serverReady := n.server.ReadyForConnections(5 * time.Second); !serverReady {
		return errors.New("server not ready for connections")
	}

	var err error
	if n.conn, err = nats.Connect("", nats.InProcessServer(n.server)); err != nil {
		return fmt.Errorf("in process connect error : %w", err)
	}

	n.LogHandler.SetNatsConn(n.conn)

	return nil
}

func (n *Nats) Shutdown() error {
	n.LogHandler.ClearNatsConn()

	var errs []error
	for _, sub := range slices.Backward(n.subs) {
		if sub != nil {
			if err := sub.Drain(); err != nil {
				errs = append(errs, fmt.Errorf("draining subscription %s: %w", sub.Subject, err))
			}
		}
	}
	if n.conn != nil {
		n.conn.Close()
	}
	if n.server != nil {
		n.server.Shutdown()
	}
	return errors.Join(errs...)
}

func (n *Nats) Subscribe(subject string, handler nats.MsgHandler) error {
	sub, err := n.conn.Subscribe(subject, handler)
	if err != nil {
		return err
	}
	n.subs = append(n.subs, sub)
	return nil
}

func (n *Nats) newServer() (err error) {
	o := &server.Options{Host: os.Getenv("NATS_HOST"), Port: 4222, NoSigs: true}
	if portEnv := os.Getenv("NATS_PORT"); portEnv != "" {
		if o.Port, err = strconv.Atoi(portEnv); err != nil {
			return fmt.Errorf("invalid NATS_PORT environment : %w", err)
		}
	}
	n.server, err = server.NewServer(o)
	return err
}

func (n *Nats) Publish(s string, bytes []byte) {
	if err := n.conn.Publish(s, bytes); err != nil {
		n.Error("Publish error", "error", err)
	}
}

func (n *Nats) PublishMsg(m *nats.Msg) {
	if err := n.conn.PublishMsg(m); err != nil {
		n.Error("PublishMsg error", "error", err)
	}
}

func (n *Nats) Reply(in *nats.Msg, payload []byte, err error) {
	if in.Reply == "" {
		return
	}
	msg := nats.NewMsg(in.Reply)
	if payload != nil {
		msg.Data = payload
	} else {
		msg.Data = []byte{}
	}
	if err != nil {
		msg.Header.Set("error", err.Error())
	}
	n.PublishMsg(msg)
}
