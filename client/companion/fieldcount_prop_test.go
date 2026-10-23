package companion

import (
	"fmt"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Feature: merlin-companion, Property 2: Parse rejects under-field messages

// TestProperty2_TooFewFields verifies that any wire payload with fewer than 8
// colon-separated fields causes all three Parse functions to return a non-nil error.
//
// **Validates: Requirements 4.3**
func TestProperty2_TooFewFields(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		fieldCount := rapid.IntRange(1, 7).Draw(t, "fieldCount")

		fields := make([]string, fieldCount)
		for i := range fields {
			fields[i] = rapid.StringMatching(`[a-zA-Z0-9]{1,10}`).Draw(t, fmt.Sprintf("field%d", i))
		}

		input := strings.Join(fields, ":")

		_, err1 := ParseCompanionEventCompanionSummoned(input)
		_, err2 := ParseCompanionEventCompanionDismissed(input)
		_, err3 := ParseCompanionEventCompanionDied(input)

		if err1 == nil {
			t.Fatalf("ParseCompanionEventCompanionSummoned(%q) expected error for %d fields, got nil", input, fieldCount)
		}
		if err2 == nil {
			t.Fatalf("ParseCompanionEventCompanionDismissed(%q) expected error for %d fields, got nil", input, fieldCount)
		}
		if err3 == nil {
			t.Fatalf("ParseCompanionEventCompanionDied(%q) expected error for %d fields, got nil", input, fieldCount)
		}
	})
}

// TestProperty2_TooManyFields verifies that any wire payload with more than 8
// colon-separated fields causes all three Parse functions to return a non-nil error.
//
// **Validates: Requirements 4.3**
func TestProperty2_TooManyFields(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		fieldCount := rapid.IntRange(9, 15).Draw(t, "fieldCount")

		fields := make([]string, fieldCount)
		for i := range fields {
			fields[i] = rapid.StringMatching(`[a-zA-Z0-9]{1,10}`).Draw(t, fmt.Sprintf("field%d", i))
		}

		input := strings.Join(fields, ":")

		_, err1 := ParseCompanionEventCompanionSummoned(input)
		_, err2 := ParseCompanionEventCompanionDismissed(input)
		_, err3 := ParseCompanionEventCompanionDied(input)

		if err1 == nil {
			t.Fatalf("ParseCompanionEventCompanionSummoned(%q) expected error for %d fields, got nil", input, fieldCount)
		}
		if err2 == nil {
			t.Fatalf("ParseCompanionEventCompanionDismissed(%q) expected error for %d fields, got nil", input, fieldCount)
		}
		if err3 == nil {
			t.Fatalf("ParseCompanionEventCompanionDied(%q) expected error for %d fields, got nil", input, fieldCount)
		}
	})
}
