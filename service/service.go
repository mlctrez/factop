package service

import (
	"context"
	kservice "github.com/kardianos/service"
	"github.com/mlctrez/servicego"
)

type Service struct {
	servicego.Defaults
	components []Component
	Nats       *Nats
	Settings   *Settings
	Factorio   *Factorio
	Rcon       *Rcon
	SoftMod    *SoftMod
	Command    *Command
	WebServer  *WebServer

	context context.Context
	cancel  context.CancelFunc
}

type Component interface {
	Start(service *Service) error
	Stop() error
}

func Run() {
	servicego.Run(New())
}

func New() servicego.Service {
	s := &Service{
		Nats:      &Nats{},
		Settings:  &Settings{},
		Factorio:  &Factorio{},
		Rcon:      &Rcon{},
		SoftMod:   &SoftMod{},
		Command:   &Command{},
		WebServer: &WebServer{},
	}
	s.components = []Component{
		s.Nats,
		s.Settings,
		s.Factorio,
		s.Rcon,
		s.SoftMod,
		s.Command,
		s.WebServer,
	}
	return s
}

func (sv *Service) Start(s kservice.Service) error {
	sv.context, sv.cancel = context.WithCancel(context.Background())
	for _, c := range sv.components {
		if err := c.Start(sv); err != nil {
			if stopErr := sv.Stop(s); stopErr != nil {
				sv.Errorf("error stopping service : %v", stopErr)
			}
			return err
		}
	}
	return nil
}

func (sv *Service) Stop(_ kservice.Service) error {
	sv.cancel()
	sv.Infof("total components %d", len(sv.components))
	for i := len(sv.components) - 1; i >= 0; i-- {
		sv.Infof("stopping component %T", sv.components[i])
		if err := sv.components[i].Stop(); err != nil {
			sv.Errorf("error stopping component %T : %s", sv.components[i], err)
		}
	}
	return nil
}
