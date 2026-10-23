// Command merlin is the Factorio companion follow plugin.
//
// Build & deploy:
//
//	CGO_ENABLED=0 go build -ldflags "-s -w" -o /tmp/merlin ./pluginexample/merlin/
//	go run cmd/main.go plugin deploy merlin 0.1.0 /tmp/merlin
//	go run cmd/main.go plugin restart merlin
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/mlctrez/factop/client/companion"
	"github.com/mlctrez/factop/client/player"
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
	stopFns    map[int]context.CancelFunc // per-player stop goroutine cancel
	dataDir    string
	ctx        plugin.Context
}

func (m *merlin) Setup(ctx plugin.Context) error {
	m.log = ctx.Logger()
	m.dataDir = ctx.DataDir()
	m.ctx = ctx
	m.companions = make(map[int]CompanionEntry)
	m.stopFns = make(map[int]context.CancelFunc)

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
	if err := ev.OnMove(m.onMove); err != nil {
		return fmt.Errorf("OnMove: %w", err)
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

// Distance thresholds for companion walking behavior (tiles).
//
// followStartDist and followStopDist form a hysteresis band: an idle companion
// only starts walking once the player is more than followStartDist away, and a
// walking companion keeps walking until it is within followStopDist. Between
// the two, an idle companion stays put, so small player movements don't make
// Merlin twitch back and forth.
const (
	followStartDist  = 5  // idle companion starts walking when farther than this
	followStopDist   = 2  // walking companion stops when closer than this
	followGiveUpDist = 30 // stop walking if too far (e.g. train travel)
)

// followAction is the decision made for a companion on a player move event.
type followAction int

const (
	followStop followAction = iota // set walking_state={walking=false}
	followHold                     // leave walking_state unchanged (idle stays idle)
	followWalk                     // walk toward the target
)

func (a followAction) String() string {
	switch a {
	case followStop:
		return "stop"
	case followHold:
		return "hold"
	case followWalk:
		return "walk"
	}
	return fmt.Sprintf("followAction(%d)", int(a))
}

// followDecision is the follow logic in pure form. dist is the distance in
// tiles from the companion to the target and walking says whether the
// companion is already walking. buildFollowCmd emits the same logic as Lua,
// because the companion's position and walking state are only known in-game.
func followDecision(dist float64, walking bool) followAction {
	if dist > followGiveUpDist || dist < followStopDist {
		return followStop
	}
	if !walking && dist <= followStartDist {
		return followHold
	}
	return followWalk
}

// buildFollowCmd returns an RCON command that makes the companion walk toward
// the given target position using walking_state, following followDecision:
// stop if close enough or too far, stay idle if idle and within
// followStartDist, otherwise walk (or keep walking) toward the target.
// Returns ("", false) if the player has no companion mapping.
func buildFollowCmd(companions map[int]CompanionEntry, mv player.Move) (string, bool) {
	entry, ok := companions[mv.Index]
	if !ok {
		return "", false
	}
	// Uses the move event coordinates as the target. The distance check runs
	// against this fixed point so Merlin stops when reaching it.
	cmd := fmt.Sprintf(`/sc local e=game.get_entity_by_unit_number(%d); if not (e and e.valid) then return end; local dx=%g-e.position.x; local dy=%g-e.position.y; local dist=math.sqrt(dx*dx+dy*dy); if dist>%d or dist<%d then e.walking_state={walking=false}; return end; if dist<=%d and not e.walking_state.walking then return end; local a=math.atan2(dx,0-dy); if a<0 then a=a+2*math.pi end; local i=math.floor(a/(math.pi/4)+0.5)%%8; local dirs={[0]=defines.direction.north,defines.direction.northeast,defines.direction.east,defines.direction.southeast,defines.direction.south,defines.direction.southwest,defines.direction.west,defines.direction.northwest}; e.walking_state={walking=true,direction=dirs[i]}`,
		entry.UnitNumber, mv.X, mv.Y, followGiveUpDist, followStopDist, followStartDist,
	)
	return cmd, true
}

// buildStopCmd returns an RCON command that stops the companion if within
// stopDist of the target position.
func buildStopCmd(unitNumber int, targetX, targetY float64) string {
	return fmt.Sprintf(`/sc local e=game.get_entity_by_unit_number(%d); if not (e and e.valid) then return end; local dx=%g-e.position.x; local dy=%g-e.position.y; if dx*dx+dy*dy<%d*%d then e.walking_state={walking=false} end`,
		unitNumber, targetX, targetY, followStopDist, followStopDist,
	)
}

func (m *merlin) onMove(mv player.Move) {
	m.mu.Lock()
	cmd, ok := buildFollowCmd(m.companions, mv)
	// Cancel any previous stop-polling goroutine for this player.
	if cancel, exists := m.stopFns[mv.Index]; exists {
		cancel()
		delete(m.stopFns, mv.Index)
	}
	if !ok {
		// No companion for this player: don't track a stop poller.
		m.mu.Unlock()
		return
	}
	entry := m.companions[mv.Index]
	ctx, cancel := context.WithCancel(context.Background())
	m.stopFns[mv.Index] = cancel
	m.mu.Unlock()
	if _, err := m.ctx.Rcon(cmd); err != nil {
		m.log.Error("follow walk failed", "player", mv.Name, "error", err)
		return
	}
	// Poll until Merlin reaches the target or a new move event supersedes this one.
	go func() {
		for range 8 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(250 * time.Millisecond):
			}
			stopCmd := buildStopCmd(entry.UnitNumber, mv.X, mv.Y)
			if _, err := m.ctx.Rcon(stopCmd); err != nil {
				return
			}
		}
	}()
}

func main() {
	plugin.Run(&merlin{})
}
