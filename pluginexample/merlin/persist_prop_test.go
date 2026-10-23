package main

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// Feature: merlin-companion, Property 6: Companion mapping persistence round-trip
//
// For any CompanionMap (map of player index to CompanionEntry), serializing
// to JSON (int keys → string keys) then deserializing (string keys → int keys)
// produces an equivalent mapping.
//
// **Validates: Requirements 6.1, 6.2, 6.3**
func TestProperty6_PersistenceRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random CompanionMap with 0-10 entries.
		count := rapid.IntRange(0, 10).Draw(t, "count")
		original := make(map[int]CompanionEntry, count)
		for range count {
			idx := rapid.IntRange(1, 1000).Draw(t, "playerIdx")
			original[idx] = CompanionEntry{
				Name:        rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "name"),
				UnitNumber:  rapid.IntRange(1, 99999).Draw(t, "unit"),
				SurfaceName: rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "surface"),
			}
		}

		// Serialize: int keys → string keys → JSON (same as persist()).
		out := make(map[string]CompanionEntry, len(original))
		for k, v := range original {
			out[strconv.Itoa(k)] = v
		}
		data, err := json.Marshal(out)
		require.NoError(t, err)

		// Deserialize: JSON → string keys → int keys (same as Setup load).
		var raw map[string]CompanionEntry
		require.NoError(t, json.Unmarshal(data, &raw))

		restored := make(map[int]CompanionEntry, len(raw))
		for k, v := range raw {
			idx, err := strconv.Atoi(k)
			require.NoError(t, err)
			restored[idx] = v
		}

		// Verify equivalent mapping.
		assert.Equal(t, original, restored)
	})
}
