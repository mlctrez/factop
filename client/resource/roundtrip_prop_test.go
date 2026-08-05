package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// Feature: lua-event-emitters, Property 4: Resource wire format round-trip

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

// TestProperty4_ResourceWireFormatRoundTrip verifies that for any valid ResourceDepleted instance,
// calling Format() then ParseResourceDepleted() produces a struct with equivalent field values
// (within floating-point tolerance of 0.1 for float64 fields).
//
// **Validates: Requirements 4.1, 4.3, 8.4**
func TestProperty4_ResourceWireFormatRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		r := ResourceDepleted{
			Name:         alphanumGen().Draw(t, "Name"),
			X:            rapid.Float64Range(-100000, 100000).Draw(t, "X"),
			Y:            rapid.Float64Range(-100000, 100000).Draw(t, "Y"),
			Amount:       rapid.Uint64().Draw(t, "Amount"),
			SurfaceName:  alphanumGen().Draw(t, "SurfaceName"),
			SurfaceIndex: rapid.IntRange(0, 100000).Draw(t, "SurfaceIndex"),
		}

		formatted := r.Format()
		parsed, err := ParseResourceDepleted(formatted)
		require.NoError(t, err)

		assert.Equal(t, r.Name, parsed.Name)
		assert.InDelta(t, r.X, parsed.X, 0.1)
		assert.InDelta(t, r.Y, parsed.Y, 0.1)
		assert.Equal(t, r.Amount, parsed.Amount)
		assert.Equal(t, r.SurfaceName, parsed.SurfaceName)
		assert.Equal(t, r.SurfaceIndex, parsed.SurfaceIndex)
	})
}
