package main

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/mlctrez/factop/client/player"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestFollowThresholdOrdering(t *testing.T) {
	assert.Less(t, followStopDist, followStartDist, "stop distance must be inside start distance for hysteresis")
	assert.Less(t, followStartDist, followGiveUpDist)
}

var followCases = []struct {
	dist    float64
	walking bool
	want    followAction
}{
	{0, false, followStop},
	{1.9, true, followStop},
	{1.9, false, followStop},
	{2, false, followHold},
	{3, false, followHold},
	{5, false, followHold},
	{5.1, false, followWalk},
	{2, true, followWalk},
	{3, true, followWalk},
	{5, true, followWalk},
	{10, false, followWalk},
	{10, true, followWalk},
	{30, false, followWalk},
	{30.1, true, followStop},
	{30.1, false, followStop},
	{100, false, followStop},
}

func TestFollowDecision(t *testing.T) {
	for _, c := range followCases {
		assert.Equal(t, c.want, followDecision(c.dist, c.walking), "dist=%g walking=%v", c.dist, c.walking)
	}
}

// Feature: merlin-companion, follow hysteresis.
// An idle companion never starts walking within followStartDist, a walking
// companion keeps walking anywhere in [followStopDist, followGiveUpDist], and
// any companion stops outside that band.
func TestPropertyFollowHysteresis(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		dist := rapid.Float64Range(0, 2*followGiveUpDist).Draw(t, "dist")
		walking := rapid.Bool().Draw(t, "walking")
		got := followDecision(dist, walking)
		switch {
		case dist < followStopDist || dist > followGiveUpDist:
			assert.Equal(t, followStop, got)
		case walking:
			assert.Equal(t, followWalk, got)
		case dist <= followStartDist:
			assert.Equal(t, followHold, got)
		default:
			assert.Equal(t, followWalk, got)
		}
	})
}

func TestBuildFollowCmdUsesStartDist(t *testing.T) {
	companions := map[int]CompanionEntry{1: {Name: "merlin", UnitNumber: 42}}
	cmd, ok := buildFollowCmd(companions, player.Move{Index: 1, X: 1.5, Y: -2})
	require.True(t, ok)
	assert.Contains(t, cmd, fmt.Sprintf("if dist<=%d and not e.walking_state.walking then return end", followStartDist))
}

// TestBuildFollowCmdLuaMatchesDecision runs the generated RCON Lua against a
// stubbed game API and checks it agrees with followDecision. Skipped when no
// Lua interpreter is installed. Factorio uses Lua 5.2, so math.atan2 is shimmed
// for newer interpreters.
func TestBuildFollowCmdLuaMatchesDecision(t *testing.T) {
	var luaBin string
	for _, name := range []string{"lua5.2", "lua5.4", "lua5.3", "lua"} {
		if p, err := exec.LookPath(name); err == nil {
			luaBin = p
			break
		}
	}
	if luaBin == "" {
		t.Skip("no lua interpreter on PATH")
	}

	var script strings.Builder
	script.WriteString(`math.atan2 = math.atan2 or function(y, x) return math.atan(y, x) end
defines = { direction = { north=0, northeast=2, east=4, southeast=6, south=8, southwest=10, west=12, northwest=14 } }
local function run(cmd, walking)
  local initial = { walking = walking }
  local e = { valid = true, position = { x = 0, y = 0 }, walking_state = initial }
  game = { get_entity_by_unit_number = function(n) return e end }
  assert(load(cmd))()
  if e.walking_state == initial then return "hold" end
  if e.walking_state.walking then return "walk" end
  return "stop"
end
`)
	// Target directly east of the companion at the given distance.
	for _, c := range followCases {
		cmd, ok := buildFollowCmd(map[int]CompanionEntry{1: {UnitNumber: 7}}, player.Move{Index: 1, X: c.dist, Y: 0})
		require.True(t, ok)
		fmt.Fprintf(&script, "print(run(%q, %v))\n", strings.TrimPrefix(cmd, "/sc "), c.walking)
	}

	out, err := exec.Command(luaBin, "-e", script.String()).CombinedOutput()
	require.NoError(t, err, string(out))
	lines := strings.Fields(string(out))
	require.Len(t, lines, len(followCases), string(out))
	for i, c := range followCases {
		assert.Equal(t, c.want.String(), lines[i], "dist=%g walking=%v", c.dist, c.walking)
	}
}
