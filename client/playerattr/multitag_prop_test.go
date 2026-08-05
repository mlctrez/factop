package playerattr

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// Feature: event-schema-codegen, Property 6: Multi-Event Tag Discrimination

// playerPayloadGen generates a random valid PlayerEvent wire payload (2 colon-separated fields: name:index).
func playerPayloadGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		length := rapid.IntRange(1, 20).Draw(t, "nameLen")
		b := make([]byte, length)
		for i := range b {
			b[i] = chars[rapid.IntRange(0, len(chars)-1).Draw(t, "nameCh")]
		}
		name := string(b)
		index := strconv.Itoa(rapid.IntRange(0, 100000).Draw(t, "index"))
		return fmt.Sprintf("%s:%s", name, index)
	})
}

// TestProperty6_PlayerEvent_TagDiscrimination verifies that for any valid wire payload,
// each tag-specific parser sets the Event field to exactly its own tag string.
//
// **Validates: Requirements 13.1, 13.3, 1.3**
func TestProperty6_PlayerEvent_TagDiscrimination(t *testing.T) {
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

	rapid.Check(t, func(t *rapid.T) {
		payload := playerPayloadGen().Draw(t, "payload")

		for _, p := range parsers {
			result, err := p.parse(payload)
			require.NoError(t, err, "parser for tag %q should succeed on valid payload", p.tag)
			assert.Equal(t, p.tag, result.Event,
				"parser for tag %q must set Event field to exactly %q", p.tag, p.tag)
		}

		// Verify each parser sets a distinct tag from the others
		results := make(map[string]string)
		for _, p := range parsers {
			result, _ := p.parse(payload)
			results[p.tag] = result.Event
		}
		for _, p := range parsers {
			for _, other := range parsers {
				if p.tag != other.tag {
					assert.NotEqual(t, results[p.tag], results[other.tag],
						"parsers for %q and %q must set different Event values", p.tag, other.tag)
				}
			}
		}
	})
}
