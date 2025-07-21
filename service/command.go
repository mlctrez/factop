package service

import (
	"errors"
	"github.com/mlctrez/bind"
	"github.com/mlctrez/factop/service/downloader"
	"github.com/nats-io/nats.go"
	"log/slog"
)

var _ bind.Startup = (*Command)(nil)

type Command struct {
	slog.Logger
	Nats     *Nats
	Factorio *Factorio
}

func (c *Command) Startup() error {
	return c.Nats.Subscribe("factop.command", c.handleCommand)
}

func (c *Command) handleCommand(msg *nats.Msg) {
	var command = string(msg.Data)

	switch command {
	case "status":
		c.Nats.Reply(msg, []byte(c.Factorio.Status()), nil)
	case "stop":
		err := c.Factorio.Shutdown()
		c.Nats.Reply(msg, []byte(c.Factorio.Status()), err)
	case "restart":
		err := c.Factorio.Restart()
		c.Nats.Reply(msg, []byte(c.Factorio.Status()), err)
	case "reset":
		err := c.Factorio.Reset()
		c.Nats.Reply(msg, []byte(c.Factorio.Status()), err)
	case "latest":
		go func() {
			err := downloader.CheckLatest()
			if err != nil {
				c.Error("downloader error", "error", err)
			}
		}()
		c.Nats.Reply(msg, []byte("download started"), nil)
	default:
		usage := "available commands: status, stop, restart, reset, latest"
		c.Nats.Reply(msg, nil, errors.New(usage))
	}

}
