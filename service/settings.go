package service

import (
	"context"
	_ "embed"
	"encoding/json"
	"github.com/mlctrez/factop/api"
	"github.com/mlctrez/servicego"
	"os"
)

var _ Component = (*Settings)(nil)

const (
	SettingsDirectory = "/opt/factorio/settings"
	FactOpSettings    = SettingsDirectory + "/factop-settings.json"
	ServerSettings    = SettingsDirectory + "/server-settings.json"
	ServerAdminList   = SettingsDirectory + "/server-adminlist.json"
	ServerBanList     = SettingsDirectory + "/server-banlist.json"
)

type Settings struct {
	servicego.DefaultLogger
}

func (s *Settings) Start(service *Service) error {
	s.Logger(service.Log())
	if err := s.setupDefaults(); err != nil {
		return err
	}
	nats := service.Nats
	impl := api.NewSettingsHandler(service.context, nats.conn, s)
	return nats.Subscribe(impl.Subject(), impl.Handler)
}

func (s *Settings) Stop() error {
	return nil
}

func (s *Settings) ServerAdmins(ctx context.Context, req *api.ServerAdminsType) (*api.ServerAdminsType, error) {
	if req.Write {
		if contents, err := json.MarshalIndent(req.Admin, "", "  "); err != nil {
			return nil, err
		} else {
			if err = os.WriteFile(ServerAdminList, contents, 0644); err != nil {
				return nil, err
			}
			return s.ServerAdmins(ctx, &api.ServerAdminsType{Write: false})
		}
	} else {
		if file, err := os.ReadFile(ServerAdminList); err != nil {
			return nil, err
		} else {
			response := &api.ServerAdminsType{Write: req.Write}
			return response, json.Unmarshal(file, &response.Admin)
		}
	}
}

func (s *Settings) setupDefaults() error {
	if err := os.MkdirAll(SettingsDirectory, 0755); err != nil {
		return err
	}
	if err := s.createDefault(FactOpSettings, factopSettings); err != nil {
		return err
	}
	if err := s.createDefault(ServerSettings, serverSettings); err != nil {
		return err
	}
	if err := s.createDefault(ServerAdminList, "[]"); err != nil {
		return err
	}
	if err := s.createDefault(ServerBanList, "[]"); err != nil {
		return err
	}
	return nil
}

func (s *Settings) createDefault(path, contents string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(path, []byte(contents+"\n"), 0644)
		}
	}
	return nil
}

//go:embed settings/factop-settings.json
var factopSettings string

//go:embed settings/server-settings.json
var serverSettings string
