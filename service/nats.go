package service

import (
	"errors"
	"fmt"
	"github.com/mlctrez/bind"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"log/slog"
	"os"
	"strconv"
	"time"
)

var _ bind.Startup = (*Nats)(nil)
var _ bind.Shutdown = (*Nats)(nil)

type Nats struct {
	slog.Logger
	server *server.Server
	conn   *nats.Conn
	subs   []*nats.Subscription
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

	return nil
}

func (n *Nats) Shutdown() error {
	for i := len(n.subs) - 1; i >= 0; i-- {
		n.unsubscribe(n.subs[i])
	}
	if n.conn != nil {
		n.conn.Close()
	}
	if n.server != nil {
		n.server.Shutdown()
	}
	return nil
}

func (n *Nats) unsubscribe(sub *nats.Subscription) {
	if sub != nil {
		if err := sub.Drain(); err != nil {
			n.Error("sub.Drain", "subject", sub.Subject, "error", err)
		}

		if err := sub.Unsubscribe(); err != nil {
			n.Error("sub.Unsubscribe", "subject", sub.Subject, "error", err)
		}
	}
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
