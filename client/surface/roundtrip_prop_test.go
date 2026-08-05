package surface

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// Feature: lua-event-emitters, Property 3: Surface wire format round-trip

// alphanumGen generates a random non-empty alphanumeric string (no colons, spaces, or brackets).
func alphanumGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"
		length := rapid.IntRange(1, 20).Draw(t, "len")
		b := make([]byte, length)
		for i := range b {
			b[i] = chars[rapid.IntRange(0, len(chars)-1).Draw(t, "ch")]
		}
		return string(b)
	})
}

// TestProperty3_SurfaceCreatedRoundTrip verifies that for any valid SurfaceEvent with tag surface_created,
// calling Format() then ParseSurfaceEventSurfaceCreated produces a struct with equivalent field values.
//
// **Validates: Requirements 3.1, 3.2, 3.3, 8.3**
func TestProperty3_SurfaceCreatedRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ev := SurfaceEvent{
			Event: "surface_created",
			Name:  alphanumGen().Draw(t, "Name"),
			Index: rapid.IntRange(0, 100000).Draw(t, "Index"),
		}

		formatted := ev.Format()
		parsed, err := ParseSurfaceEventSurfaceCreated(formatted)
		require.NoError(t, err)

		assert.Equal(t, ev.Name, parsed.Name)
		assert.Equal(t, ev.Index, parsed.Index)
	})
}

// TestProperty3_SurfaceDeletedRoundTrip verifies that for any valid SurfaceEvent with tag surface_deleted,
// calling Format() then ParseSurfaceEventSurfaceDeleted produces a struct with equivalent field values.
//
// **Validates: Requirements 3.1, 3.2, 3.3, 8.3**
func TestProperty3_SurfaceDeletedRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ev := SurfaceEvent{
			Event: "surface_deleted",
			Name:  alphanumGen().Draw(t, "Name"),
			Index: rapid.IntRange(0, 100000).Draw(t, "Index"),
		}

		formatted := ev.Format()
		parsed, err := ParseSurfaceEventSurfaceDeleted(formatted)
		require.NoError(t, err)

		assert.Equal(t, ev.Name, parsed.Name)
		assert.Equal(t, ev.Index, parsed.Index)
	})
}
