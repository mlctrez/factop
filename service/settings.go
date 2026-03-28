package service

import (
	_ "embed"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"regexp"
	"slices"

	"github.com/mlctrez/bind"
)

var _ bind.Startup = (*Settings)(nil)

const (
	SettingsDirectory  = "/opt/factorio/settings"
	FactOpSettingsPath = SettingsDirectory + "/factop-settings.json"
	ServerSettings     = SettingsDirectory + "/server-settings.json"
	MapGenSettings     = SettingsDirectory + "/map-gen-settings.json"
	ServerAdminList    = SettingsDirectory + "/server-adminlist.json"
	ServerBanList      = SettingsDirectory + "/server-banlist.json"
)

type Settings struct {
	*slog.Logger `bind:"-"`
	Data         FactOpSettings `bind:"-"`
}

type FactOpSettings struct {
	Port            string
	SaveLocation    string
	FactorioVersion string
}

func (s *Settings) Startup() error {
	if err := os.MkdirAll(SettingsDirectory, 0755); err != nil {
		return err
	}
	if err := s.createDefault(FactOpSettingsPath, factopSettings); err != nil {
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

	if err := s.Load(); err != nil {
		return err
	}

	if s.Data.FactorioVersion == "" || !versionRegex.MatchString(s.Data.FactorioVersion) {
		if s.Data.FactorioVersion == "" {
			s.Info("no factorio version configured, picking latest")
		} else {
			s.Warn("incorrect factorio version configured, picking latest", "version", s.Data.FactorioVersion)
		}
		latest, err := s.detectLatest()
		if err != nil {
			return err
		}
		s.Data.FactorioVersion = latest
		if err = s.Save(); err != nil {
			return err
		}
	}

	return nil
}

var versionRegex = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

func (s *Settings) detectLatest() (string, error) {
	entries, err := os.ReadDir("/opt/factorio")
	if err != nil {
		return "", err
	}
	var versions []string
	for _, entry := range entries {
		if entry.IsDir() && versionRegex.MatchString(entry.Name()) {
			versions = append(versions, entry.Name())
		}
	}
	if len(versions) == 0 {
		return "", errors.New("no local factorio versions found in /opt/factorio")
	}
	slices.Sort(versions)
	return versions[len(versions)-1], nil
}

func (s *Settings) Load() error {
	data, err := os.ReadFile(FactOpSettingsPath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.Data)
}

func (s *Settings) Save() error {
	data, err := json.MarshalIndent(s.Data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(FactOpSettingsPath, data, 0644)
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
