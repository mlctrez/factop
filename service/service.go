package service

import (
	"context"
	kservice "github.com/kardianos/service"
	"github.com/mlctrez/bind"
	"github.com/mlctrez/servicego"
	"log/slog"
)

type Service struct {
	servicego.Defaults
	binder  bind.Binder
	Context context.Context
	Cancel  context.CancelFunc
	//holder *ContextHolder
}

func Run() {
	servicego.Run(&Service{})
}

//type Component interface {
//	bind.Startup
//	bind.Shutdown
//}

func (sv *Service) Start(s kservice.Service) error {

	//sv.holder = &ContextHolder{}
	sv.Context, sv.Cancel = context.WithCancel(context.Background())

	sv.binder = bind.New()

	components := []any{
		slog.New(&LogHandler{Logger: sv.Log()}),
		sv.Context,
		&Nats{},
		&Settings{},
		&SoftMod{},
		&Factorio{},
		&Rcon{},
		&Command{},
		&WebServer{},
		&UDPBridge{},
	}

	return sv.binder.Add(components...)
}

func (sv *Service) Stop(_ kservice.Service) error {
	sv.Cancel()
	if sv.binder != nil {
		sv.binder.Shutdown()
	}
	return nil
}
