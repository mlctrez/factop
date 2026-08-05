package main

import (
	"fmt"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Feature: lua-codegen, Property 4: Unrecognized Annotation Key Detection

// recognizedKeySet is the full set of recognized key patterns for gen:lua annotations.
// Exact matches: event, guard, iterate, loop_var, tag, nth_tick, include_cause
// Prefix matches: field.*, default.*, fallback.*
var recognizedExactKeys = []string{
	"event", "guard", "iterate", "loop_var", "tag", "nth_tick", "include_cause",
}

var recognizedPrefixKeys = []string{
	"field.", "default.", "fallback.",
}

// genRecognizedKey generates a valid recognized key for gen:lua annotations.
func genRecognizedKey() *rapid.Generator[string] {
	return rapid.Custom[string](func(t *rapid.T) string {
		strategy := rapid.IntRange(0, 1).Draw(t, "keyStrategy")
		switch strategy {
		case 0:
			// Pick an exact key.
			idx := rapid.IntRange(0, len(recognizedExactKeys)-1).Draw(t, "exactKeyIdx")
			return recognizedExactKeys[idx]
		default:
			// Pick a prefix key and append a valid suffix.
			idx := rapid.IntRange(0, len(recognizedPrefixKeys)-1).Draw(t, "prefixKeyIdx")
			suffix := rapid.StringMatching(`[a-z][a-z0-9_]{0,15}`).Draw(t, "keySuffix")
			return recognizedPrefixKeys[idx] + suffix
		}
	})
}

// genUnrecognizedKey generates a key that does NOT match any recognized pattern.
func genUnrecognizedKey() *rapid.Generator[string] {
	return rapid.Custom[string](func(t *rapid.T) string {
		// Generate a key that:
		// - Is not in the exact recognized set
		// - Does not start with any recognized prefix
		// Use a prefix that no recognized key uses.
		badPrefixes := []string{
			"unknown_", "foo_", "bar_", "xyz_", "custom_",
			"emit_", "source_", "output_", "gen_", "lua_",
		}
		prefixIdx := rapid.IntRange(0, len(badPrefixes)-1).Draw(t, "badPrefixIdx")
		suffix := rapid.StringMatching(`[a-z][a-z0-9_]{0,10}`).Draw(t, "badKeySuffix")
		return badPrefixes[prefixIdx] + suffix
	})
}

// TestProperty4_UnrecognizedKeyDetection verifies that for any set of gen:lua
// annotation lines containing one or more unrecognized keys, the validator
// reports an error identifying the unrecognized key name and the struct name.
//
// **Validates: Requirements 1.10**
func TestProperty4_UnrecognizedKeyDetection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a struct name.
		structName := rapid.StringMatching(`[A-Z][A-Za-z]{2,15}`).Draw(t, "structName")

		// Generate a valid tag so other validations pass.
		tag := rapid.StringMatching(`[a-z][a-z0-9_]{1,10}`).Draw(t, "tag")

		// Build a base LuaSchema with valid recognized keys.
		numRecognized := rapid.IntRange(0, 5).Draw(t, "numRecognized")
		rawKeys := make([]string, 0, numRecognized+1)

		// Always include "tag" as it's required.
		rawKeys = append(rawKeys, "tag")

		for i := 0; i < numRecognized; i++ {
			key := genRecognizedKey().Draw(t, fmt.Sprintf("recognizedKey_%d", i))
			rawKeys = append(rawKeys, key)
		}

		// Inject one or more unrecognized keys.
		numBad := rapid.IntRange(1, 3).Draw(t, "numBadKeys")
		var injectedBadKeys []string
		for i := 0; i < numBad; i++ {
			badKey := genUnrecognizedKey().Draw(t, fmt.Sprintf("badKey_%d", i))
			injectedBadKeys = append(injectedBadKeys, badKey)
			rawKeys = append(rawKeys, badKey)
		}

		// Build the schema with valid fields so only key validation triggers.
		schema := EventSchema{
			StructName: structName,
			Tags:       []string{tag},
			Fields: []FieldSchema{
				{Name: "Placeholder", Type: "string", Position: 0},
			},
			Lua: &LuaSchema{
				Tag:      tag,
				Fields:   make(map[string]string),
				Defaults: make(map[string]string),
				RawKeys:  rawKeys,
			},
		}

		// Call ValidateLua and verify it returns an error.
		err := ValidateLua(schema)
		if err == nil {
			t.Fatalf("expected error for unrecognized keys %v, but validation passed", injectedBadKeys)
		}

		errMsg := err.Error()

		// The error must identify the struct name.
		if !strings.Contains(errMsg, structName) {
			t.Fatalf("error %q should contain struct name %q", errMsg, structName)
		}

		// The error must identify at least one of the unrecognized keys.
		foundBadKey := false
		for _, badKey := range injectedBadKeys {
			if strings.Contains(errMsg, badKey) {
				foundBadKey = true
				break
			}
		}
		if !foundBadKey {
			t.Fatalf("error %q should contain at least one unrecognized key from %v", errMsg, injectedBadKeys)
		}
	})
}

// Feature: lua-codegen, Property 2: Tag Validation Pattern

// TestProperty2_TagValidationPattern verifies that the tag validation accepts
// a string if and only if it is non-empty and matches ^[a-z0-9_]+$. Strings
// containing uppercase letters, spaces, special characters, or empty strings
// are rejected with a validation error.
//
// **Validates: Requirements 1.6**
func TestProperty2_TagValidationPattern(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random string from a wide character set including valid
		// and invalid characters (letters, digits, punctuation, spaces).
		tag := rapid.OneOf(
			// Valid: only lowercase letters, digits, underscores (non-empty)
			rapid.StringMatching(`[a-z0-9_]{1,20}`),
			// Invalid: contains uppercase, spaces, or special characters
			rapid.StringMatching(`[a-zA-Z0-9_ !@#$%^&*]{0,20}`),
			// Explicitly test empty string
			rapid.Just(""),
		).Draw(t, "tag")

		// Determine expected validity: non-empty AND matches ^[a-z0-9_]+$
		expectedValid := tag != "" && tagPattern.MatchString(tag)

		// Build a minimal valid EventSchema with the tag under test in LuaSchema.
		schema := EventSchema{
			StructName: "TestStruct",
			Tags:       []string{"test_tag"},
			Fields: []FieldSchema{
				{Name: "Value", Type: "string", Position: 0},
			},
			Lua: &LuaSchema{
				Tag:      tag,
				Fields:   make(map[string]string),
				Defaults: make(map[string]string),
				RawKeys:  []string{"tag"},
			},
		}

		err := ValidateLua(schema)

		if expectedValid {
			// Valid tags should pass validation (no error from tag check).
			// Other validations might still trigger, but NOT for tag format.
			if err != nil && strings.Contains(err.Error(), "tag") && strings.Contains(err.Error(), "invalid") {
				t.Fatalf("tag %q should be valid but got tag validation error: %v", tag, err)
			}
			if err != nil && strings.Contains(err.Error(), "missing required tag") {
				t.Fatalf("tag %q should be valid but got missing tag error: %v", tag, err)
			}
		} else {
			// Invalid tags should produce a validation error about the tag.
			if err == nil {
				t.Fatalf("tag %q should be invalid but validation passed", tag)
			}
			errMsg := err.Error()
			// Error must mention tag-related issue.
			if !strings.Contains(errMsg, "tag") {
				t.Fatalf("tag %q is invalid but error %q does not mention tag", tag, errMsg)
			}
		}
	})
}

// Feature: lua-codegen, Property 3: NthTick Interval Validation

// TestProperty3_NthTickIntervalValidation verifies that for any integer value
// specified as an nth_tick interval, validation accepts it if and only if it is
// in the range [1, 3600]. Values outside this range (including negative numbers
// and values > 3600) produce a validation error. NthTick=0 means "disabled" and
// is not validated by the validator.
//
// **Validates: Requirements 1.9, 5.3**
func TestProperty3_NthTickIntervalValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random integer in a wide range, excluding 0 (disabled).
		// Use two strategies: pick from valid range or invalid range.
		interval := rapid.OneOf(
			rapid.IntRange(-1000, -1),
			rapid.IntRange(1, 10000),
		).Draw(t, "interval")

		// Determine expected validity: valid iff in [1, 3600].
		expectedValid := interval >= 1 && interval <= 3600

		// Build a minimal valid schema with the NthTick value.
		structName := rapid.StringMatching(`[A-Z][A-Za-z]{2,15}`).Draw(t, "structName")
		tag := rapid.StringMatching(`[a-z][a-z0-9_]{1,10}`).Draw(t, "tag")

		schema := EventSchema{
			StructName: structName,
			Tags:       []string{tag},
			Fields: []FieldSchema{
				{Name: "Placeholder", Type: "string", Position: 0},
			},
			Lua: &LuaSchema{
				Tag:      tag,
				NthTick:  interval,
				Fields:   make(map[string]string),
				Defaults: make(map[string]string),
				RawKeys:  []string{"tag", "nth_tick"},
			},
		}

		err := ValidateLua(schema)

		if expectedValid && err != nil {
			t.Fatalf("expected nth_tick=%d to be valid, got error: %v", interval, err)
		}
		if !expectedValid && err == nil {
			t.Fatalf("expected nth_tick=%d to be invalid, but validation passed", interval)
		}
		if !expectedValid && err != nil {
			// Verify error message contains struct name and the interval value.
			errMsg := err.Error()
			if !strings.Contains(errMsg, structName) {
				t.Fatalf("error %q should contain struct name %q", errMsg, structName)
			}
			if !strings.Contains(errMsg, fmt.Sprintf("%d", interval)) {
				t.Fatalf("error %q should contain interval value %d", errMsg, interval)
			}
		}
	})
}
