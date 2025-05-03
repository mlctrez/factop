package service

import (
	"errors"
	"fmt"
	"github.com/mlctrez/servicego"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"os"
	"strconv"
	"time"
)

var _ Component = (*Nats)(nil)

type Nats struct {
	servicego.DefaultLogger
	server *server.Server
	conn   *nats.Conn
	subs   []*nats.Subscription
}

func (n *Nats) Start(l *Service) error {
	n.Logger(l.Log())
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

func (n *Nats) Stop() error {
	for i := len(n.subs) - 1; i >= 0; i-- {
		sub := n.subs[i]
		if sub == nil {
			continue
		}
		if err := sub.Unsubscribe(); err != nil {
			n.Errorf("error unsubscribing from %s : %v", sub.Subject, err)
		}
	}
	n.server.Shutdown()
	return nil
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
		n.Errorf("error publishing to %s : %v", s, err)
	}
}

func (n *Nats) PublishMsg(m *nats.Msg) {
	if err := n.conn.PublishMsg(m); err != nil {
		n.Errorf("error publishing to %s : %v", m.Subject, err)
	}
}
