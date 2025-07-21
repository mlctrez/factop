package service

import (
	_ "embed"
	"github.com/mlctrez/bind"
	"os"
)

var _ bind.Startup = (*Settings)(nil)

const (
	SettingsDirectory = "/opt/factorio/settings"
	FactOpSettings    = SettingsDirectory + "/factop-settings.json"
	ServerSettings    = SettingsDirectory + "/server-settings.json"
	MapGenSettings    = SettingsDirectory + "/map-gen-settings.json"
	ServerAdminList   = SettingsDirectory + "/server-adminlist.json"
	ServerBanList     = SettingsDirectory + "/server-banlist.json"
)

type Settings struct{}

func (s *Settings) Startup() error {
	if err := os.MkdirAll(SettingsDirectory, 0755); err != nil {
		return err
	}
	if err := s.createDefault(FactOpSettings, factopSettings); err != nil {
		return err
	}
	if err := s.createDefault(ServerSettings, serverSettings); err != nil {
		return err
	}
	if err := s.createDefault(MapGenSettings, mapGenSettings); err != nil {
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

//go:embed settings/map-gen-settings.json
var mapGenSettings string
