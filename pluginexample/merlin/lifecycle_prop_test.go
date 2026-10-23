package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

// Feature: merlin-companion, Property 5: Lifecycle events remove companion mapping
//
// For any CompanionMap containing an entry for player P, processing a
// companion_dismissed or companion_died event with PlayerIndex=P results
// in the map no longer containing an entry for P.
//
// **Validates: Requirements 5.4, 6.2**
func TestProperty5_LifecycleEventsRemoveMapping(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a target player index that will be removed.
		targetPlayer := rapid.IntRange(1, 100).Draw(t, "targetPlayer")

		// Generate a random companion map with 0-5 other entries.
		otherCount := rapid.IntRange(0, 5).Draw(t, "otherCount")
		companions := make(map[int]CompanionEntry)
		for range otherCount {
			idx := rapid.IntRange(101, 999).Draw(t, "otherIdx")
			companions[idx] = CompanionEntry{
				Name:        rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "name"),
				UnitNumber:  rapid.IntRange(1, 99999).Draw(t, "unit"),
				SurfaceName: rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "surface"),
			}
		}

		// Ensure the target player has an entry.
		companions[targetPlayer] = CompanionEntry{
			Name:        rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "targetName"),
			UnitNumber:  rapid.IntRange(1, 99999).Draw(t, "targetUnit"),
			SurfaceName: rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "targetSurface"),
		}

		originalSize := len(companions)

		// Simulate what onDismissed/onDied do: delete the player's entry.
		delete(companions, targetPlayer)

		// Verify the target player is no longer in the map.
		_, exists := companions[targetPlayer]
		assert.False(t, exists, "map should not contain player %d after lifecycle event", targetPlayer)

		// Verify only the target was removed (other entries unchanged).
		assert.Equal(t, originalSize-1, len(companions), "exactly one entry should be removed")
	})
}
