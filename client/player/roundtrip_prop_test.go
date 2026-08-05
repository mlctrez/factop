package player

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// Feature: event-schema-codegen, Property 1: Parse-Format Round Trip

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

// TestProperty1_ParseFormatRoundTrip verifies that for any valid Move instance,
// calling Format() then ParseMove() produces a struct with equivalent field values
// (within floating-point tolerance of 0.1 for float64 fields).
//
// **Validates: Requirements 5.6, 4.5, 4.6, 4.7, 5.2, 5.3, 5.4, 5.5**
func TestProperty1_ParseFormatRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		m := Move{
			Name:         alphanumGen().Draw(t, "Name"),
			Index:        rapid.IntRange(0, 100000).Draw(t, "Index"),
			X:            rapid.Float64Range(-100000, 100000).Draw(t, "X"),
			Y:            rapid.Float64Range(-100000, 100000).Draw(t, "Y"),
			SurfaceName:  alphanumGen().Draw(t, "SurfaceName"),
			SurfaceIndex: rapid.IntRange(0, 100000).Draw(t, "SurfaceIndex"),
		}

		formatted := m.Format()
		parsed, err := ParseMove(formatted)
		require.NoError(t, err)

		assert.Equal(t, m.Name, parsed.Name)
		assert.Equal(t, m.Index, parsed.Index)
		assert.InDelta(t, m.X, parsed.X, 0.1)
		assert.InDelta(t, m.Y, parsed.Y, 0.1)
		assert.Equal(t, m.SurfaceName, parsed.SurfaceName)
		assert.Equal(t, m.SurfaceIndex, parsed.SurfaceIndex)
	})
}
