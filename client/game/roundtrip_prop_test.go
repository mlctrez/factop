package game

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// Feature: lua-event-emitters, Property 5: Game tick wire format round-trip

// TestProperty5_GameTickRoundTrip verifies that for any valid tick value (uint64),
// formatting it with Format() and then parsing with ParseTickEvent produces a struct
// with an equivalent tick value.
//
// **Validates: Requirements 5.1, 5.2, 8.5**
func TestProperty5_GameTickRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tick := TickEvent{
			Tick: rapid.Uint64().Draw(t, "Tick"),
		}

		formatted := tick.Format()
		parsed, err := ParseTickEvent(formatted)
		require.NoError(t, err)

		assert.Equal(t, tick.Tick, parsed.Tick)
	})
}
