package companion

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

// Feature: merlin-companion, Property 3: Wire payload size stays under UDP limit

// boundedAlphanumGen generates a random non-empty alphanumeric string (no colons) up to maxLen bytes.
func boundedAlphanumGen(maxLen int) *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-."
		length := rapid.IntRange(1, maxLen).Draw(t, "len")
		b := make([]byte, length)
		for i := range b {
			b[i] = chars[rapid.IntRange(0, len(chars)-1).Draw(t, "ch")]
		}
		return string(b)
	})
}

// TestProperty3_WirePayloadSizeUnderUDPLimit verifies that for any valid CompanionEvent
// with Name ≤ 50 bytes, PlayerName ≤ 50 bytes, and SurfaceName ≤ 50 bytes,
// calling Format() produces a string whose byte length is less than 512.
//
// **Validates: Requirements 9.2**
func TestProperty3_WirePayloadSizeUnderUDPLimit(t *testing.T) {
	tags := []string{"companion_summoned", "companion_dismissed", "companion_died"}

	rapid.Check(t, func(t *rapid.T) {
		ev := CompanionEvent{
			Event:        tags[rapid.IntRange(0, len(tags)-1).Draw(t, "tagIdx")],
			Name:         boundedAlphanumGen(50).Draw(t, "Name"),
			PlayerIndex:  rapid.IntRange(0, 100000).Draw(t, "PlayerIndex"),
			PlayerName:   boundedAlphanumGen(50).Draw(t, "PlayerName"),
			UnitNumber:   rapid.IntRange(0, 100000000).Draw(t, "UnitNumber"),
			X:            rapid.Float64Range(-1000000, 1000000).Draw(t, "X"),
			Y:            rapid.Float64Range(-1000000, 1000000).Draw(t, "Y"),
			SurfaceName:  boundedAlphanumGen(50).Draw(t, "SurfaceName"),
			SurfaceIndex: rapid.IntRange(0, 100000).Draw(t, "SurfaceIndex"),
		}

		formatted := ev.Format()
		assert.Less(t, len(formatted), 512, "wire payload must be under 512 bytes, got %d", len(formatted))
	})
}
