// Command merlin is the Factorio companion plugin.
//
// The softmod (softmod/factop/companion.lua) owns Merlin in-game: it creates
// the character on /merlin, makes him follow his owner every few ticks with
// on_nth_tick, and emits companion_summoned/dismissed/died events. This plugin
// only tracks which player owns which companion and logs lifecycle changes;
// it does not drive movement (no per-move RCON).
//
// Build & deploy:
//
//	CGO_ENABLED=0 go build -ldflags "-s -w" -o /tmp/merlin ./pluginexample/merlin/
//	go run cmd/main.go plugin deploy merlin 2.0.1 /tmp/merlin
//	go run cmd/main.go plugin restart merlin
package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/mlctrez/factop/client/companion"
	"github.com/mlctrez/factop/plugin"
)

// CompanionEntry is the per-player companion state persisted to disk.
type CompanionEntry struct {
	Name        string `json:"name"`
	UnitNumber  int    `json:"unit_number"`
	SurfaceName string `json:"surface_name"`
}

type merlin struct {
	log        *slog.Logger
	mu         sync.Mutex
	companions map[int]CompanionEntry
	dataDir    string
}

func (m *merlin) Setup(ctx plugin.Context) error {
	m.log = ctx.Logger()
	m.dataDir = ctx.DataDir()
	m.companions = make(map[int]CompanionEntry)

	data, err := os.ReadFile(filepath.Join(m.dataDir, "companions.json"))
	if err != nil {
		if !os.IsNotExist(err) {
			m.log.Error("failed to read companions.json", "error", err)
		}
		return nil
	}

	var raw map[string]CompanionEntry
	if err := json.Unmarshal(data, &raw); err != nil {
		m.log.Error("malformed companions.json", "error", err)
		return nil
	}
	for k, v := range raw {
		idx, err := strconv.Atoi(k)
		if err != nil {
			m.log.Error("invalid player index key in companions.json", "key", k, "error", err)
			continue
		}
		m.companions[idx] = v
	}
	m.log.Info("companions loaded", "count", len(m.companions))
	return nil
}

func (m *merlin) Run(ctx plugin.Context) error {
	ev := ctx.Events()

	if err := ev.OnCompanionSummoned(m.onSummoned); err != nil {
		return fmt.Errorf("OnCompanionSummoned: %w", err)
	}
	if err := ev.OnCompanionDismissed(m.onDismissed); err != nil {
		return fmt.Errorf("OnCompanionDismissed: %w", err)
	}
	if err := ev.OnCompanionDied(m.onDied); err != nil {
		return fmt.Errorf("OnCompanionDied: %w", err)
	}

	m.log.Info("merlin running")
	<-ctx.Done()
	m.log.Info("merlin shutting down")
	return nil
}

func (m *merlin) onSummoned(ev companion.CompanionEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.companions[ev.PlayerIndex]; exists {
		m.log.Warn("replacing existing companion mapping", "player", ev.PlayerName, "index", ev.PlayerIndex)
	}
	m.companions[ev.PlayerIndex] = CompanionEntry{
		Name:        ev.Name,
		UnitNumber:  ev.UnitNumber,
		SurfaceName: ev.SurfaceName,
	}
	m.persist()
	m.log.Info("companion summoned", "player", ev.PlayerName, "unit", ev.UnitNumber)
}

func (m *merlin) onDismissed(ev companion.CompanionEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.companions, ev.PlayerIndex)
	m.persist()
	m.log.Info("companion dismissed", "player", ev.PlayerName)
}

func (m *merlin) onDied(ev companion.CompanionEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.companions, ev.PlayerIndex)
	m.persist()
	m.log.Info("companion died", "player", ev.PlayerName)
}

// persist writes the companion map to companions.json atomically.
// Must be called with m.mu already held.
func (m *merlin) persist() {
	out := make(map[string]CompanionEntry, len(m.companions))
	for k, v := range m.companions {
		out[strconv.Itoa(k)] = v
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		m.log.Error("failed to marshal companions", "error", err)
		return
	}
	path := filepath.Join(m.dataDir, "companions.json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		m.log.Error("failed to write companions temp file", "error", err)
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		m.log.Error("failed to rename companions file", "error", err)
	}
}

func main() {
	plugin.Run(&merlin{})
}
