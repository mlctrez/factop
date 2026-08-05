package main

import (
	"fmt"
	"strings"

	"github.com/mlctrez/factop/plugin"
)

// World runs short Lua snippets in Factorio via RCON (/sc).
type World struct {
	ctx plugin.Context
}

func NewWorld(ctx plugin.Context) *World {
	return &World{ctx: ctx}
}

// GamePrint shows a message to all players.
func (w *World) GamePrint(msg string) error {
	cmd := fmt.Sprintf("/sc game.print(%s)", luaString(msg))
	_, err := w.ctx.Rcon(cmd)
	return err
}

// PlayerPrint shows a message to one player by index (1-based).
// If index is 0 or invalid, falls back to game.print.
func (w *World) PlayerPrint(index int, msg string) error {
	if index <= 0 {
		return w.GamePrint(msg)
	}
	cmd := fmt.Sprintf(
		"/sc local p=game.players[%d]; if p and p.valid then p.print(%s) else game.print(%s) end",
		index, luaString(msg), luaString(msg),
	)
	_, err := w.ctx.Rcon(cmd)
	return err
}

// SpawnEnemy places an enemy force entity near a position.
func (w *World) SpawnEnemy(surface string, x, y float64, name string) error {
	if surface == "" {
		surface = "nauvis"
	}
	if name == "" {
		name = "small-biter"
	}
	cmd := fmt.Sprintf(
		"/sc local s=game.surfaces[%s]; if s and s.valid then s.create_entity{name=%s, position={%g,%g}, force=game.forces.enemy} end",
		luaString(surface), luaString(name), x, y,
	)
	_, err := w.ctx.Rcon(cmd)
	return err
}

// SpawnEnemies places count enemies in a small ring around (x, y).
func (w *World) SpawnEnemies(surface string, x, y float64, name string, count int) error {
	if count < 1 {
		count = 1
	}
	// Offsets keep units from stacking on one tile.
	offsets := [][2]float64{
		{3, 0}, {-3, 0}, {0, 3}, {0, -3},
		{2.5, 2.5}, {-2.5, 2.5}, {2.5, -2.5}, {-2.5, -2.5},
	}
	for i := 0; i < count; i++ {
		off := offsets[i%len(offsets)]
		if err := w.SpawnEnemy(surface, x+off[0], y+off[1], name); err != nil {
			return err
		}
	}
	return nil
}

// SpawnEnemiesNearPlayer spawns enemies around the first connected player.
func (w *World) SpawnEnemiesNearPlayer(name string, count int) error {
	if name == "" {
		name = "small-biter"
	}
	if count < 1 {
		count = 1
	}
	// Build a Lua snippet that places count enemies around player 1 (or first valid).
	cmd := fmt.Sprintf(`/sc
local name=%s
local n=%d
local p
for _,pl in pairs(game.connected_players) do p=pl break end
if not (p and p.valid) then return end
local s=p.surface
local x,y=p.position.x,p.position.y
local offs={{3,0},{-3,0},{0,3},{0,-3},{2.5,2.5},{-2.5,2.5}}
for i=1,n do
  local o=offs[((i-1)%%#offs)+1]
  s.create_entity{name=name, position={x+o[1], y+o[2]}, force=game.forces.enemy}
end
`, luaString(name), count)
	// collapse to single line for RCON
	cmd = strings.ReplaceAll(cmd, "\n", " ")
	_, err := w.ctx.Rcon(cmd)
	return err
}

// AnnounceToAll prints a story line for every connected player and game log.
func (w *World) AnnounceToAll(msg string) error {
	cmd := fmt.Sprintf(
		"/sc game.print(%s); for _,p in pairs(game.connected_players) do if p.valid then p.print(%s) end end",
		luaString(msg), luaString(msg),
	)
	_, err := w.ctx.Rcon(cmd)
	return err
}

// luaString quotes a Go string as a Lua single-quoted literal.
func luaString(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('\'')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '\'':
			b.WriteString(`\'`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('\'')
	return b.String()
}
