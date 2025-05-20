package service

import (
	"context"
	"errors"
	"github.com/mlctrez/factop/api"
	"github.com/mlctrez/factop/service/downloader"
	"github.com/mlctrez/servicego"
)

var _ Component = (*Command)(nil)

type Command struct {
	servicego.DefaultLogger
	Factorio *Factorio
}

func (c *Command) Start(s *Service) error {
	c.Logger(s.Log())
	c.Factorio = s.Factorio
	impl := api.NewCommandHandler(s.context, s.Nats.conn, c)
	return s.Nats.Subscribe(impl.Subject(), impl.Handler)
}

func (c *Command) Stop() error {
	return nil
}

func (c *Command) Execute(ctx context.Context, req *api.CommandPayload) (*api.CommandResult, error) {
	switch req.Command {
	case "status":
		return &api.CommandResult{Payload: c.Factorio.Status()}, nil
	case "stop":
		return &api.CommandResult{}, c.Factorio.Stop()
	case "restart":
		return &api.CommandResult{}, c.Factorio.Restart()
	case "reset":
		return &api.CommandResult{}, c.Factorio.Reset()
	case "latest":
		go func() {
			err := downloader.CheckLatest()
			if err != nil {
				c.Errorf("downloader error: %v", err)
			}
		}()
		return &api.CommandResult{}, nil

	default:
		usage := "available commands: status, stop, restart, reset"
		return &api.CommandResult{}, errors.New(usage)
	}
}
