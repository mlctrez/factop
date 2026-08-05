package entity

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// Feature: event-schema-codegen, Property 6: Multi-Event Tag Discrimination

// entityPayloadGen generates a random valid EntityEvent wire payload (7 or 8 colon-separated fields).
func entityPayloadGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		genStr := func(label string) string {
			length := rapid.IntRange(1, 15).Draw(t, label+"Len")
			b := make([]byte, length)
			for i := range b {
				b[i] = chars[rapid.IntRange(0, len(chars)-1).Draw(t, label+"Ch")]
			}
			return string(b)
		}

		name := genStr("name")
		x := strconv.FormatFloat(rapid.Float64Range(-10000, 10000).Draw(t, "x"), 'f', 1, 64)
		y := strconv.FormatFloat(rapid.Float64Range(-10000, 10000).Draw(t, "y"), 'f', 1, 64)
		unitNumber := strconv.FormatUint(rapid.Uint64Range(0, 999999).Draw(t, "unitNumber"), 10)
		surfaceName := genStr("surfaceName")
		surfaceIndex := strconv.Itoa(rapid.IntRange(0, 100).Draw(t, "surfaceIndex"))
		playerIndex := strconv.Itoa(rapid.IntRange(0, 100).Draw(t, "playerIndex"))

		payload := fmt.Sprintf("%s:%s:%s:%s:%s:%s:%s", name, x, y, unitNumber, surfaceName, surfaceIndex, playerIndex)

		// Sometimes include an 8th optional Cause field
		if rapid.Bool().Draw(t, "hasCause") {
			cause := genStr("cause")
			payload += ":" + cause
		}

		return payload
	})
}

// TestProperty6_EntityEvent_TagDiscrimination verifies that for any valid wire payload,
// each tag-specific parser sets the Event field to exactly its own tag string.
//
// **Validates: Requirements 13.1, 13.3, 1.3**
func TestProperty6_EntityEvent_TagDiscrimination(t *testing.T) {
	type parserEntry struct {
		tag   string
		parse func(string) (EntityEvent, error)
	}

	parsers := []parserEntry{
		{"entity_died", ParseEntityEventEntityDied},
		{"entity_built", ParseEntityEventEntityBuilt},
		{"entity_mined", ParseEntityEventEntityMined},
	}

	rapid.Check(t, func(t *rapid.T) {
		payload := entityPayloadGen().Draw(t, "payload")

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
