// Command rpgdemo is the Factorio RPG story plugin (First Night slice).
//
// Build & deploy:
//
//	CGO_ENABLED=0 go build -ldflags "-s -w" -o /tmp/rpgdemo ./pluginexample/rpgdemo/
//	go run cmd/main.go plugin deploy rpgdemo 0.2.0 /tmp/rpgdemo
//	go run cmd/main.go plugin restart rpgdemo
package main

import (
	"fmt"
	"log/slog"
	"math"
	"strings"

	"github.com/mlctrez/factop/client/entity"
	"github.com/mlctrez/factop/client/player"
	"github.com/mlctrez/factop/client/playerattr"
	"github.com/mlctrez/factop/plugin"
)

// Distance in tiles from the first recorded position before "explore" fires.
const exploreDistance = 40.0

// How many small biters to spawn when industry is first established.
const siegeBiterCount = 3

// Enemy prototype names that count toward siege resolution.
var siegeEnemies = map[string]bool{
	"small-biter":      true,
	"medium-biter":     true,
	"big-biter":        true,
	"behemoth-biter":   true,
	"small-spitter":    true,
	"medium-spitter":   true,
	"big-spitter":      true,
	"behemoth-spitter": true,
}

// Entity names that count as "industry" for the first pressure beat.
var industryEntities = map[string]bool{
	"stone-furnace":         true,
	"steel-furnace":         true,
	"electric-furnace":      true,
	"burner-mining-drill":   true,
	"electric-mining-drill": true,
}

type rpgDemo struct {
	log   *slog.Logger
	store *Store
	world *World
}

func (r *rpgDemo) Setup(ctx plugin.Context) error {
	r.log = ctx.Logger()
	store, err := OpenStore(ctx.DataDir())
	if err != nil {
		return fmt.Errorf("open campaign store: %w", err)
	}
	r.store = store
	r.world = NewWorld(ctx)

	snap := store.Snapshot()
	r.log.Info("campaign loaded",
		"stage", snap.Stage,
		"flags", len(snap.Flags),
		"kills", snap.KillCount,
		"spawns", snap.SpawnCount,
		"data_dir", ctx.DataDir(),
	)
	return nil
}

func (r *rpgDemo) Run(ctx plugin.Context) error {
	ev := ctx.Events()

	if err := ev.OnPlayerJoined(r.onJoined); err != nil {
		return fmt.Errorf("OnPlayerJoined: %w", err)
	}
	if err := ev.OnMove(r.onMove); err != nil {
		return fmt.Errorf("OnMove: %w", err)
	}
	if err := ev.OnEntityBuilt(r.onEntityBuilt); err != nil {
		return fmt.Errorf("OnEntityBuilt: %w", err)
	}
	if err := ev.OnEntityDied(r.onEntityDied); err != nil {
		return fmt.Errorf("OnEntityDied: %w", err)
	}
	if err := ev.OnPlayerDied(r.onPlayerDied); err != nil {
		return fmt.Errorf("OnPlayerDied: %w", err)
	}

	// Resume feedback for players already in-game when the plugin starts.
	r.announceStatus()

	r.log.Info("rpgdemo running")
	<-ctx.Done()
	r.log.Info("rpgdemo shutting down")
	return nil
}

func (r *rpgDemo) announceStatus() {
	snap := r.store.Snapshot()
	var msg string
	switch {
	case snap.Flags[FlagFirstNightDone]:
		msg = fmt.Sprintf("[RPG] Director online. First night complete (stage=%s). Expand the factory — more awaits later.", snap.Stage)
	case snap.Flags[FlagSiegeSpawned]:
		need := snap.SpawnCount - snap.KillCount
		if need < 1 {
			need = siegeBiterCount
		}
		msg = fmt.Sprintf("[RPG] Director online. Siege in progress — defeat the remaining biters (%d left to track).", need)
		// Re-issue a pack so a mid-session restart is still completable.
		if err := r.world.SpawnEnemiesNearPlayer("small-biter", siegeBiterCount); err != nil {
			r.log.Error("siege re-spawn failed", "error", err)
		} else {
			_ = r.store.Update(func(s *State) (bool, error) {
				s.SpawnCount = s.KillCount + siegeBiterCount
				return true, nil
			})
		}
	case snap.Flags[FlagIndustry]:
		msg = "[RPG] Director online. Industry stands — wait for the dark to answer."
	case snap.Flags[FlagExplored]:
		msg = "[RPG] Director online. Build a furnace or mining drill to claim this land."
	default:
		msg = "[RPG] Director online. Explore, then build a furnace or mining drill."
	}
	if err := r.world.AnnounceToAll(msg); err != nil {
		r.log.Error("status announce failed", "error", err)
	}
}

func (r *rpgDemo) onJoined(ev playerattr.PlayerEvent) {
	var msg string
	err := r.store.Update(func(s *State) (bool, error) {
		if s.Flags[FlagWelcomed] {
			msg = fmt.Sprintf("[RPG] Welcome back, %s. Stage: %s.", ev.Name, s.Stage)
			return false, nil
		}
		s.Flags[FlagWelcomed] = true
		if s.Stage == "" || s.Stage == StageIntro {
			s.Stage = StageIntro
		}
		msg = fmt.Sprintf(
			"[RPG] Welcome, %s. Explore the land, then build a furnace or mining drill — the night is not empty.",
			ev.Name,
		)
		return true, nil
	})
	if err != nil {
		r.log.Error("join state update failed", "error", err)
		return
	}
	if msg == "" {
		return
	}
	if err := r.world.PlayerPrint(ev.Index, msg); err != nil {
		r.log.Error("join print failed", "error", err)
	}
}

func (r *rpgDemo) onMove(mv player.Move) {
	var msg string
	err := r.store.Update(func(s *State) (bool, error) {
		if s.Flags[FlagExplored] || s.Flags[FlagFirstNightDone] {
			return false, nil
		}
		if !s.HasOrigin {
			s.OriginX = mv.X
			s.OriginY = mv.Y
			s.HasOrigin = true
			return true, nil
		}
		dx := mv.X - s.OriginX
		dy := mv.Y - s.OriginY
		if math.Hypot(dx, dy) < exploreDistance {
			return false, nil
		}
		s.Flags[FlagExplored] = true
		if s.Stage == StageIntro {
			s.Stage = StageExplore
		}
		msg = "[RPG] Something watches from the trees. Industry will draw it closer."
		return true, nil
	})
	if err != nil {
		r.log.Error("move state update failed", "error", err)
		return
	}
	if msg == "" {
		return
	}
	if err := r.world.PlayerPrint(mv.Index, msg); err != nil {
		r.log.Error("explore print failed", "error", err)
	}
}

func (r *rpgDemo) onEntityBuilt(ev entity.EntityEvent) {
	if !industryEntities[ev.Name] {
		return
	}

	var msg string
	var spawn bool
	var surface string
	var x, y float64
	var playerIdx int

	err := r.store.Update(func(s *State) (bool, error) {
		if s.Flags[FlagIndustry] || s.Flags[FlagFirstNightDone] {
			return false, nil
		}
		s.Flags[FlagIndustry] = true
		s.Flags[FlagSiegeSpawned] = true
		s.Stage = StageSiege
		s.SpawnCount = siegeBiterCount
		s.KillCount = 0
		msg = fmt.Sprintf(
			"[RPG] Your %s draws attention. Something answers from the dark — defend yourself!",
			prettyEntity(ev.Name),
		)
		spawn = true
		surface = ev.SurfaceName
		x, y = ev.X, ev.Y
		playerIdx = ev.PlayerIndex
		return true, nil
	})
	if err != nil {
		r.log.Error("industry state update failed", "error", err)
		return
	}
	if msg != "" {
		if err := r.world.PlayerPrint(playerIdx, msg); err != nil {
			r.log.Error("industry print failed", "error", err)
		}
	}
	if !spawn {
		return
	}
	if err := r.world.SpawnEnemies(surface, x, y, "small-biter", siegeBiterCount); err != nil {
		r.log.Error("siege spawn failed", "error", err)
		return
	}
	r.log.Info("siege spawned", "near", ev.Name, "x", x, "y", y, "count", siegeBiterCount)
}

func (r *rpgDemo) onEntityDied(ev entity.EntityEvent) {
	if !siegeEnemies[ev.Name] {
		return
	}

	var msg string
	err := r.store.Update(func(s *State) (bool, error) {
		if !s.Flags[FlagSiegeSpawned] || s.Flags[FlagFirstNightDone] {
			return false, nil
		}
		s.KillCount++
		need := s.SpawnCount
		if need < 1 {
			need = siegeBiterCount
		}
		if s.KillCount < need {
			msg = fmt.Sprintf("[RPG] Enemy down (%d/%d).", s.KillCount, need)
			return true, nil
		}
		// Victory path for First Night.
		s.Flags[FlagFirstNightDone] = true
		s.Stage = StageDone
		msg = "[RPG] The first night is yours. The factory holds — for now. Expand, research, and prepare: larger problems will come."
		return true, nil
	})
	if err != nil {
		r.log.Error("kill state update failed", "error", err)
		return
	}
	if msg == "" {
		return
	}
	if err := r.world.GamePrint(msg); err != nil {
		r.log.Error("kill print failed", "error", err)
	}
}

func (r *rpgDemo) onPlayerDied(ev playerattr.PlayerEvent) {
	var msg string
	err := r.store.Update(func(s *State) (bool, error) {
		// Death during siege ends the first night the hard way (once).
		if s.Flags[FlagSiegeSpawned] && !s.Flags[FlagFirstNightDone] {
			s.Flags[FlagFirstDeath] = true
			s.Flags[FlagFirstNightDone] = true
			s.Stage = StageDone
			msg = fmt.Sprintf(
				"[RPG] %s falls. The first night ends in blood — rebuild, or the dark will return.",
				ev.Name,
			)
			return true, nil
		}
		if s.Flags[FlagFirstDeath] {
			return false, nil
		}
		s.Flags[FlagFirstDeath] = true
		msg = fmt.Sprintf("[RPG] %s has fallen. The factory must go on.", ev.Name)
		return true, nil
	})
	if err != nil {
		r.log.Error("death state update failed", "error", err)
		return
	}
	if msg == "" {
		return
	}
	if err := r.world.GamePrint(msg); err != nil {
		r.log.Error("death print failed", "error", err)
	}
}

func prettyEntity(name string) string {
	return strings.ReplaceAll(name, "-", " ")
}

func main() {
	plugin.Run(&rpgDemo{})
}
