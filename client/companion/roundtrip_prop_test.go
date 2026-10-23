package companion

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// Feature: merlin-companion, Property 1: Companion event wire-format round-trip

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

func companionEventGen(tag string) *rapid.Generator[CompanionEvent] {
	return rapid.Custom(func(t *rapid.T) CompanionEvent {
		return CompanionEvent{
			Event:        tag,
			Name:         alphanumGen().Draw(t, "Name"),
			PlayerIndex:  rapid.IntRange(0, 100000).Draw(t, "PlayerIndex"),
			PlayerName:   alphanumGen().Draw(t, "PlayerName"),
			UnitNumber:   rapid.IntRange(1, 100000).Draw(t, "UnitNumber"),
			X:            rapid.Float64Range(-100000, 100000).Draw(t, "X"),
			Y:            rapid.Float64Range(-100000, 100000).Draw(t, "Y"),
			SurfaceName:  alphanumGen().Draw(t, "SurfaceName"),
			SurfaceIndex: rapid.IntRange(0, 100000).Draw(t, "SurfaceIndex"),
		}
	})
}

// TestProperty1_CompanionSummoned_RoundTrip verifies Format→Parse round-trip for companion_summoned.
//
// **Validates: Requirements 4.1, 4.2, 4.5, 9.1**
func TestProperty1_CompanionSummoned_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ev := companionEventGen("companion_summoned").Draw(t, "event")

		formatted := ev.Format()
		parsed, err := ParseCompanionEventCompanionSummoned(formatted)
		require.NoError(t, err)

		assert.Equal(t, ev.Name, parsed.Name)
		assert.Equal(t, ev.PlayerIndex, parsed.PlayerIndex)
		assert.Equal(t, ev.PlayerName, parsed.PlayerName)
		assert.Equal(t, ev.UnitNumber, parsed.UnitNumber)
		assert.InDelta(t, ev.X, parsed.X, 0.1)
		assert.InDelta(t, ev.Y, parsed.Y, 0.1)
		assert.Equal(t, ev.SurfaceName, parsed.SurfaceName)
		assert.Equal(t, ev.SurfaceIndex, parsed.SurfaceIndex)
	})
}

// TestProperty1_CompanionDismissed_RoundTrip verifies Format→Parse round-trip for companion_dismissed.
//
// **Validates: Requirements 4.1, 4.2, 4.5, 9.1**
func TestProperty1_CompanionDismissed_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ev := companionEventGen("companion_dismissed").Draw(t, "event")

		formatted := ev.Format()
		parsed, err := ParseCompanionEventCompanionDismissed(formatted)
		require.NoError(t, err)

		assert.Equal(t, ev.Name, parsed.Name)
		assert.Equal(t, ev.PlayerIndex, parsed.PlayerIndex)
		assert.Equal(t, ev.PlayerName, parsed.PlayerName)
		assert.Equal(t, ev.UnitNumber, parsed.UnitNumber)
		assert.InDelta(t, ev.X, parsed.X, 0.1)
		assert.InDelta(t, ev.Y, parsed.Y, 0.1)
		assert.Equal(t, ev.SurfaceName, parsed.SurfaceName)
		assert.Equal(t, ev.SurfaceIndex, parsed.SurfaceIndex)
	})
}

// TestProperty1_CompanionDied_RoundTrip verifies Format→Parse round-trip for companion_died.
//
// **Validates: Requirements 4.1, 4.2, 4.5, 9.1**
func TestProperty1_CompanionDied_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ev := companionEventGen("companion_died").Draw(t, "event")

		formatted := ev.Format()
		parsed, err := ParseCompanionEventCompanionDied(formatted)
		require.NoError(t, err)

		assert.Equal(t, ev.Name, parsed.Name)
		assert.Equal(t, ev.PlayerIndex, parsed.PlayerIndex)
		assert.Equal(t, ev.PlayerName, parsed.PlayerName)
		assert.Equal(t, ev.UnitNumber, parsed.UnitNumber)
		assert.InDelta(t, ev.X, parsed.X, 0.1)
		assert.InDelta(t, ev.Y, parsed.Y, 0.1)
		assert.Equal(t, ev.SurfaceName, parsed.SurfaceName)
		assert.Equal(t, ev.SurfaceIndex, parsed.SurfaceIndex)
	})
}
