package playerattr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// Feature: lua-event-emitters, Property 6: PlayerAttr wire format round-trip

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

// TestProperty6_PlayerAttrRoundTrip verifies that for any valid player lifecycle event data
// (name, index), formatting it with Format() and then parsing with the corresponding
// tag-specific parser produces a struct with equivalent field values.
//
// **Validates: Requirements 6.1, 6.2, 6.3, 6.4, 6.7, 8.6**
func TestProperty6_PlayerAttrRoundTrip(t *testing.T) {
	type parserEntry struct {
		tag   string
		parse func(string) (PlayerEvent, error)
	}

	parsers := []parserEntry{
		{"player_joined", ParsePlayerEventPlayerJoined},
		{"player_left", ParsePlayerEventPlayerLeft},
		{"player_died", ParsePlayerEventPlayerDied},
		{"player_respawned", ParsePlayerEventPlayerRespawned},
	}

	for _, p := range parsers {
		p := p
		t.Run(p.tag, func(t *testing.T) {
			rapid.Check(t, func(t *rapid.T) {
				input := PlayerEvent{
					Event: p.tag,
					Name:  alphanumGen().Draw(t, "Name"),
					Index: rapid.IntRange(0, 100000).Draw(t, "Index"),
				}

				formatted := input.Format()
				parsed, err := p.parse(formatted)
				require.NoError(t, err)

				assert.Equal(t, p.tag, parsed.Event)
				assert.Equal(t, input.Name, parsed.Name)
				assert.Equal(t, input.Index, parsed.Index)
			})
		})
	}
}
