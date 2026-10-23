package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mlctrez/factop/client/player"
	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

// Feature: merlin-companion, Property 4: Follow command fires if and only if mapping exists

// alphanumGen generates a random non-empty alphanumeric string.
func alphanumGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		length := rapid.IntRange(1, 20).Draw(t, "len")
		b := make([]byte, length)
		for i := range b {
			b[i] = chars[rapid.IntRange(0, len(chars)-1).Draw(t, "ch")]
		}
		return string(b)
	})
}

// TestProperty4_FollowCmdIfAndOnlyIfMappingExists verifies that buildFollowCmd
// returns a command iff the companion map contains the move's player index, and that
// the command references the correct unit number and player position.
//
// **Validates: Requirements 5.1, 5.2**
func TestProperty4_FollowCmdIfAndOnlyIfMappingExists(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a companion map with 0-5 entries
		mapSize := rapid.IntRange(0, 5).Draw(t, "mapSize")
		companions := make(map[int]CompanionEntry, mapSize)
		for i := 0; i < mapSize; i++ {
			idx := rapid.IntRange(1, 100).Draw(t, fmt.Sprintf("mapKey%d", i))
			companions[idx] = CompanionEntry{
				Name:        alphanumGen().Draw(t, fmt.Sprintf("entryName%d", i)),
				UnitNumber:  rapid.IntRange(1, 999999).Draw(t, fmt.Sprintf("unitNum%d", i)),
				SurfaceName: alphanumGen().Draw(t, fmt.Sprintf("surface%d", i)),
			}
		}

		// Generate a random Move event
		mv := player.Move{
			Name:         alphanumGen().Draw(t, "moveName"),
			Index:        rapid.IntRange(1, 100).Draw(t, "moveIndex"),
			X:            rapid.Float64Range(-100, 100).Draw(t, "moveX"),
			Y:            rapid.Float64Range(-100, 100).Draw(t, "moveY"),
			SurfaceName:  alphanumGen().Draw(t, "moveSurface"),
			SurfaceIndex: rapid.IntRange(0, 10).Draw(t, "moveSurfaceIdx"),
		}

		cmd, ok := buildFollowCmd(companions, mv)
		entry, exists := companions[mv.Index]

		// Property: command produced iff mapping exists
		assert.Equal(t, exists, ok, "follow cmd issued iff mapping contains move.Index")

		if exists {
			// Verify correct unit number in command
			assert.True(t, strings.Contains(cmd, fmt.Sprintf("get_entity_by_unit_number(%d)", entry.UnitNumber)),
				"command should contain unit number %d", entry.UnitNumber)
			// Verify move event coords are used as target
			assert.True(t, strings.Contains(cmd, fmt.Sprintf("%g-e.position.x", mv.X)),
				"command should reference move X position")
			assert.True(t, strings.Contains(cmd, fmt.Sprintf("%g-e.position.y", mv.Y)),
				"command should reference move Y position")
		} else {
			assert.Empty(t, cmd)
		}
	})
}
