package player

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Feature: event-schema-codegen, Property 4: No-Panic Safety

// TestProperty4_NoPanicSafety verifies that for any byte sequence (including empty,
// random bytes, strings with only separators, and strings with no separators), the
// generated ParseMove function does NOT panic and returns either a valid struct or
// a non-nil error.
//
// **Validates: Requirements 12.4**
func TestProperty4_NoPanicSafety(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a diverse set of inputs: empty, random ASCII, random bytes,
		// only separators, no separators, and strings with [tag] prefix patterns.
		category := rapid.IntRange(0, 6).Draw(t, "category")

		var input string

		switch category {
		case 0:
			// Empty string
			input = ""

		case 1:
			// Random ASCII string (general random string)
			input = rapid.String().Draw(t, "randomString")

		case 2:
			// Random bytes including non-printable characters
			rawBytes := rapid.SliceOf(rapid.Byte()).Draw(t, "randomBytes")
			input = string(rawBytes)

		case 3:
			// String with only colon separators
			n := rapid.IntRange(0, 20).Draw(t, "colonCount")
			input = strings.Repeat(":", n)

		case 4:
			// String with no colons (no separators)
			chars := rapid.SliceOfN(
				rapid.ByteRange(32, 57), // printable ASCII excluding ':'(58)
				0, 50,
			).Draw(t, "noColonBytes")
			input = string(chars)

		case 5:
			// String with [tag] prefix pattern followed by random content
			tag := rapid.StringMatching(`[a-z_]+`).Draw(t, "tag")
			rest := rapid.String().Draw(t, "rest")
			input = "[" + tag + "] " + rest

		case 6:
			// String with [move] prefix and random payload
			payload := rapid.String().Draw(t, "payload")
			input = "[move] " + payload
		}

		// Call ParseMove inside a deferred recover to catch panics
		var didPanic bool
		var result Move
		var err error

		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			result, err = ParseMove(input)
		}()

		// Assert no panic occurred
		if didPanic {
			t.Fatalf("ParseMove panicked on input %q", input)
		}

		// Assert that either err is nil (valid parse) or err is non-nil (parse error)
		// — i.e., the function always returns cleanly. This is trivially true if we
		// reached here without panic, but we explicitly check both return paths.
		if err == nil {
			// Valid parse: result should be usable (it's a zero-value or populated struct)
			_ = result
		}
		// If err != nil, that's a valid parse error — also acceptable.
	})
}
