package player

import (
	"fmt"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Feature: event-schema-codegen, Property 3: Field Count Validation

// TestProperty3_TooFewFields verifies that for any wire payload with fewer than 6
// colon-separated fields, ParseMove returns a non-nil error containing the expected
// and actual field counts.
//
// **Validates: Requirements 4.3, 12.1, 12.3**
func TestProperty3_TooFewFields(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a field count from 1 to 5 (fewer than the required 6).
		// We use 1 as minimum because strings.Join with 0 fields produces ""
		// and strings.Split("", ":") yields 1 field, not 0.
		fieldCount := rapid.IntRange(1, 5).Draw(t, "fieldCount")

		// Generate random field values.
		fields := make([]string, fieldCount)
		for i := 0; i < fieldCount; i++ {
			fields[i] = rapid.StringMatching(`[a-zA-Z0-9._-]{1,20}`).Draw(t, fmt.Sprintf("field%d", i))
		}

		input := strings.Join(fields, ":")

		// The actual field count seen by the parser is len(strings.Split(input, ":")),
		// which equals fieldCount for non-empty fields joined by ":".
		actualFieldCount := len(strings.Split(input, ":"))

		_, err := ParseMove(input)
		if err == nil {
			t.Fatalf("ParseMove(%q) expected error for %d fields (fewer than 6), got nil", input, actualFieldCount)
		}

		// Verify error message contains expected count "6" and actual count.
		errMsg := err.Error()
		expectedStr := "6"
		actualStr := fmt.Sprintf("%d", actualFieldCount)

		if !strings.Contains(errMsg, expectedStr) {
			t.Fatalf("error message %q does not contain expected count %q", errMsg, expectedStr)
		}
		if !strings.Contains(errMsg, actualStr) {
			t.Fatalf("error message %q does not contain actual count %q", errMsg, actualStr)
		}
	})
}

// TestProperty3_TooManyFields verifies that for any wire payload with more than 6
// colon-separated fields, ParseMove returns a non-nil error containing the expected
// and actual field counts.
//
// **Validates: Requirements 4.3, 12.1, 12.3**
func TestProperty3_TooManyFields(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a field count from 7 to 15 (more than the required 6).
		fieldCount := rapid.IntRange(7, 15).Draw(t, "fieldCount")

		// Generate random field values.
		fields := make([]string, fieldCount)
		for i := 0; i < fieldCount; i++ {
			fields[i] = rapid.StringMatching(`[a-zA-Z0-9._-]{1,20}`).Draw(t, fmt.Sprintf("field%d", i))
		}

		input := strings.Join(fields, ":")

		_, err := ParseMove(input)
		if err == nil {
			t.Fatalf("ParseMove(%q) expected error for %d fields (more than 6), got nil", input, fieldCount)
		}

		// Verify error message contains expected count "6" and actual count.
		errMsg := err.Error()
		expectedStr := "6"
		actualStr := fmt.Sprintf("%d", fieldCount)

		if !strings.Contains(errMsg, expectedStr) {
			t.Fatalf("error message %q does not contain expected count %q", errMsg, expectedStr)
		}
		if !strings.Contains(errMsg, actualStr) {
			t.Fatalf("error message %q does not contain actual count %q", errMsg, actualStr)
		}
	})
}
