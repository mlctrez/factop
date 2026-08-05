package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// Feature: lua-event-emitters, Property 1: Entity wire format round-trip

// alphanumGen generates a random non-empty alphanumeric string (no colons).
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

// TestProperty1_EntityDied_RoundTrip verifies that for any valid EntityEvent with tag entity_died,
// calling Format() then ParseEntityEventEntityDied() produces a struct with equivalent field values.
//
// **Validates: Requirements 1.1, 1.2, 1.4, 1.6, 1.7, 1.8, 1.9, 1.11, 8.1**
func TestProperty1_EntityDied_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ev := EntityEvent{
			Event:        "entity_died",
			Name:         alphanumGen().Draw(t, "Name"),
			X:            rapid.Float64Range(-100000, 100000).Draw(t, "X"),
			Y:            rapid.Float64Range(-100000, 100000).Draw(t, "Y"),
			UnitNumber:   rapid.Uint64().Draw(t, "UnitNumber"),
			SurfaceName:  alphanumGen().Draw(t, "SurfaceName"),
			SurfaceIndex: rapid.IntRange(0, 100000).Draw(t, "SurfaceIndex"),
			PlayerIndex:  rapid.IntRange(0, 100000).Draw(t, "PlayerIndex"),
			Cause:        alphanumGen().Draw(t, "Cause"),
		}

		formatted := ev.Format()
		parsed, err := ParseEntityEventEntityDied(formatted)
		require.NoError(t, err)

		assert.Equal(t, ev.Name, parsed.Name)
		assert.InDelta(t, ev.X, parsed.X, 0.1)
		assert.InDelta(t, ev.Y, parsed.Y, 0.1)
		assert.Equal(t, ev.UnitNumber, parsed.UnitNumber)
		assert.Equal(t, ev.SurfaceName, parsed.SurfaceName)
		assert.Equal(t, ev.SurfaceIndex, parsed.SurfaceIndex)
		assert.Equal(t, ev.PlayerIndex, parsed.PlayerIndex)
		assert.Equal(t, ev.Cause, parsed.Cause)
	})
}

// TestProperty1_EntityDied_EmptyCause_RoundTrip verifies entity_died with an empty cause field
// (nil cause scenario) round-trips correctly.
//
// **Validates: Requirements 1.1, 1.8, 8.1**
func TestProperty1_EntityDied_EmptyCause_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ev := EntityEvent{
			Event:        "entity_died",
			Name:         alphanumGen().Draw(t, "Name"),
			X:            rapid.Float64Range(-100000, 100000).Draw(t, "X"),
			Y:            rapid.Float64Range(-100000, 100000).Draw(t, "Y"),
			UnitNumber:   rapid.Uint64().Draw(t, "UnitNumber"),
			SurfaceName:  alphanumGen().Draw(t, "SurfaceName"),
			SurfaceIndex: rapid.IntRange(0, 100000).Draw(t, "SurfaceIndex"),
			PlayerIndex:  rapid.IntRange(0, 100000).Draw(t, "PlayerIndex"),
			Cause:        "",
		}

		formatted := ev.Format()
		parsed, err := ParseEntityEventEntityDied(formatted)
		require.NoError(t, err)

		assert.Equal(t, ev.Name, parsed.Name)
		assert.InDelta(t, ev.X, parsed.X, 0.1)
		assert.InDelta(t, ev.Y, parsed.Y, 0.1)
		assert.Equal(t, ev.UnitNumber, parsed.UnitNumber)
		assert.Equal(t, ev.SurfaceName, parsed.SurfaceName)
		assert.Equal(t, ev.SurfaceIndex, parsed.SurfaceIndex)
		assert.Equal(t, ev.PlayerIndex, parsed.PlayerIndex)
		assert.Equal(t, "", parsed.Cause)
	})
}

// TestProperty1_EntityBuilt_RoundTrip verifies that for any valid EntityEvent with tag entity_built,
// calling Format() then ParseEntityEventEntityBuilt() produces a struct with equivalent field values.
//
// **Validates: Requirements 1.2, 1.6, 1.9, 1.11, 8.1**
func TestProperty1_EntityBuilt_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ev := EntityEvent{
			Event:        "entity_built",
			Name:         alphanumGen().Draw(t, "Name"),
			X:            rapid.Float64Range(-100000, 100000).Draw(t, "X"),
			Y:            rapid.Float64Range(-100000, 100000).Draw(t, "Y"),
			UnitNumber:   rapid.Uint64().Draw(t, "UnitNumber"),
			SurfaceName:  alphanumGen().Draw(t, "SurfaceName"),
			SurfaceIndex: rapid.IntRange(0, 100000).Draw(t, "SurfaceIndex"),
			PlayerIndex:  rapid.IntRange(0, 100000).Draw(t, "PlayerIndex"),
			Cause:        "",
		}

		formatted := ev.Format()
		parsed, err := ParseEntityEventEntityBuilt(formatted)
		require.NoError(t, err)

		assert.Equal(t, ev.Name, parsed.Name)
		assert.InDelta(t, ev.X, parsed.X, 0.1)
		assert.InDelta(t, ev.Y, parsed.Y, 0.1)
		assert.Equal(t, ev.UnitNumber, parsed.UnitNumber)
		assert.Equal(t, ev.SurfaceName, parsed.SurfaceName)
		assert.Equal(t, ev.SurfaceIndex, parsed.SurfaceIndex)
		assert.Equal(t, ev.PlayerIndex, parsed.PlayerIndex)
	})
}

// TestProperty1_EntityMined_RoundTrip verifies that for any valid EntityEvent with tag entity_mined,
// calling Format() then ParseEntityEventEntityMined() produces a struct with equivalent field values.
//
// **Validates: Requirements 1.4, 1.6, 1.9, 1.11, 8.1**
func TestProperty1_EntityMined_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ev := EntityEvent{
			Event:        "entity_mined",
			Name:         alphanumGen().Draw(t, "Name"),
			X:            rapid.Float64Range(-100000, 100000).Draw(t, "X"),
			Y:            rapid.Float64Range(-100000, 100000).Draw(t, "Y"),
			UnitNumber:   rapid.Uint64().Draw(t, "UnitNumber"),
			SurfaceName:  alphanumGen().Draw(t, "SurfaceName"),
			SurfaceIndex: rapid.IntRange(0, 100000).Draw(t, "SurfaceIndex"),
			PlayerIndex:  rapid.IntRange(0, 100000).Draw(t, "PlayerIndex"),
			Cause:        "",
		}

		formatted := ev.Format()
		parsed, err := ParseEntityEventEntityMined(formatted)
		require.NoError(t, err)

		assert.Equal(t, ev.Name, parsed.Name)
		assert.InDelta(t, ev.X, parsed.X, 0.1)
		assert.InDelta(t, ev.Y, parsed.Y, 0.1)
		assert.Equal(t, ev.UnitNumber, parsed.UnitNumber)
		assert.Equal(t, ev.SurfaceName, parsed.SurfaceName)
		assert.Equal(t, ev.SurfaceIndex, parsed.SurfaceIndex)
		assert.Equal(t, ev.PlayerIndex, parsed.PlayerIndex)
	})
}
