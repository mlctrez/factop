package service

import (
	"context"
	"fmt"
	"log/slog"

	kservice "github.com/kardianos/service"
	"github.com/mlctrez/bind"
	"github.com/mlctrez/servicego"
)

type Service struct {
	servicego.Defaults
	binder      bind.Binder
	Context     context.Context
	CancelCause context.CancelCauseFunc
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
	sv.Context, sv.CancelCause = context.WithCancelCause(context.Background())

	sv.binder = bind.New()

	logHandler := &LogHandler{Logger: sv.Log()}

	components := []any{
		slog.New(logHandler),
		logHandler,
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
	sv.CancelCause(fmt.Errorf("service stop requested"))
	if sv.binder != nil {
		sv.binder.Shutdown()
	}
	return nil
}
