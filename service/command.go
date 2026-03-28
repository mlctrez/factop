package service

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/mlctrez/bind"
	"github.com/mlctrez/factop/service/downloader"
	"github.com/nats-io/nats.go"
)

var _ bind.Startup = (*Command)(nil)

type Command struct {
	slog.Logger
	Nats     *Nats
	Factorio *Factorio
	Settings *Settings
}

func (c *Command) Startup() error {
	return c.Nats.Subscribe("factop.command", c.handleCommand)
}

func (c *Command) handleCommand(msg *nats.Msg) {
	parts := strings.Fields(string(msg.Data))
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "status":
		c.Nats.Reply(msg, []byte(c.Factorio.Status()), nil)
	case "start":
		err := c.Factorio.Start()
		c.Nats.Reply(msg, []byte(c.Factorio.Status()), err)
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
	case "list-versions":
		versions, err := versionDirs()
		if err != nil {
			c.Nats.Reply(msg, nil, err)
			return
		}
		slices.Sort(versions)
		c.Nats.Reply(msg, []byte(strings.Join(versions, "\n")), nil)
	case "set-version":
		if len(parts) < 2 {
			c.Nats.Reply(msg, nil, errors.New("usage: set-version <version>"))
			return
		}
		newVersion := parts[1]
		// verify it exists
		if _, err := os.Stat(filepath.Join("/opt/factorio", newVersion)); os.IsNotExist(err) {
			c.Nats.Reply(msg, nil, fmt.Errorf("version %s not found", newVersion))
			return
		}
		c.Settings.Data.FactorioVersion = newVersion
		if err := c.Settings.Save(); err != nil {
			c.Nats.Reply(msg, nil, err)
			return
		}
		err := c.Factorio.Restart()
		c.Nats.Reply(msg, []byte(fmt.Sprintf("version set to %s and restarting: %s", newVersion, c.Factorio.Status())), err)
	default:
		usage := "available commands: status, start, stop, restart, reset, latest, list-versions, set-version <version>"
		c.Nats.Reply(msg, nil, errors.New(usage))
	}
}
