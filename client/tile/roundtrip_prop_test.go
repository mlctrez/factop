package tile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// Feature: lua-event-emitters, Property 2: Tile wire format round-trip

// alphanumGen generates a random non-empty alphanumeric string (no colons, spaces, or brackets).
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

// TestProperty2_TileWireFormatRoundTrip_TileBuilt verifies that for any valid TileEvent,
// calling Format() then ParseTileEventTileBuilt() produces a struct with equivalent field values
// (within floating-point tolerance of 0.1 for float64 fields).
//
// **Validates: Requirements 2.1, 2.3, 2.5, 8.2**
func TestProperty2_TileWireFormatRoundTrip_TileBuilt(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		te := TileEvent{
			Event:        "tile_built",
			Name:         alphanumGen().Draw(t, "Name"),
			X:            rapid.Float64Range(-100000, 100000).Draw(t, "X"),
			Y:            rapid.Float64Range(-100000, 100000).Draw(t, "Y"),
			SurfaceName:  alphanumGen().Draw(t, "SurfaceName"),
			SurfaceIndex: rapid.IntRange(0, 100000).Draw(t, "SurfaceIndex"),
			PlayerIndex:  rapid.IntRange(0, 100000).Draw(t, "PlayerIndex"),
		}

		formatted := te.Format()
		parsed, err := ParseTileEventTileBuilt(formatted)
		require.NoError(t, err)

		assert.Equal(t, te.Name, parsed.Name)
		assert.InDelta(t, te.X, parsed.X, 0.1)
		assert.InDelta(t, te.Y, parsed.Y, 0.1)
		assert.Equal(t, te.SurfaceName, parsed.SurfaceName)
		assert.Equal(t, te.SurfaceIndex, parsed.SurfaceIndex)
		assert.Equal(t, te.PlayerIndex, parsed.PlayerIndex)
	})
}

// TestProperty2_TileWireFormatRoundTrip_TileMined verifies that for any valid TileEvent,
// calling Format() then ParseTileEventTileMined() produces a struct with equivalent field values
// (within floating-point tolerance of 0.1 for float64 fields).
//
// **Validates: Requirements 2.1, 2.3, 2.5, 8.2**
func TestProperty2_TileWireFormatRoundTrip_TileMined(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		te := TileEvent{
			Event:        "tile_mined",
			Name:         alphanumGen().Draw(t, "Name"),
			X:            rapid.Float64Range(-100000, 100000).Draw(t, "X"),
			Y:            rapid.Float64Range(-100000, 100000).Draw(t, "Y"),
			SurfaceName:  alphanumGen().Draw(t, "SurfaceName"),
			SurfaceIndex: rapid.IntRange(0, 100000).Draw(t, "SurfaceIndex"),
			PlayerIndex:  rapid.IntRange(0, 100000).Draw(t, "PlayerIndex"),
		}

		formatted := te.Format()
		parsed, err := ParseTileEventTileMined(formatted)
		require.NoError(t, err)

		assert.Equal(t, te.Name, parsed.Name)
		assert.InDelta(t, te.X, parsed.X, 0.1)
		assert.InDelta(t, te.Y, parsed.Y, 0.1)
		assert.Equal(t, te.SurfaceName, parsed.SurfaceName)
		assert.Equal(t, te.SurfaceIndex, parsed.SurfaceIndex)
		assert.Equal(t, te.PlayerIndex, parsed.PlayerIndex)
	})
}
