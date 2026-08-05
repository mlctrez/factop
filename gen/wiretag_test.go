package main

import (
	"fmt"
	"testing"

	"pgregory.net/rapid"
)

// Feature: event-schema-codegen, Property 2: Wire Tag Parsing Correctness

// TestParseWireTag_ValidStrings verifies that for any string matching the grammar
// position[,modifier] where position is a non-negative integer or "-" and modifier
// is "tag" or "optional", parseWireTag extracts the correct position, isTag, and
// optional values.
// **Validates: Requirements 2.1, 2.2, 2.3, 2.4, 1.2, 1.3, 1.4**
func TestParseWireTag_ValidStrings(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Decide which category of valid wire tag to generate
		category := rapid.IntRange(0, 4).Draw(t, "category")

		var input string
		var expectedPos int
		var expectedIsTag bool
		var expectedOptional bool

		switch category {
		case 0:
			// Non-negative integer position with no modifier: "N"
			pos := rapid.IntRange(0, 10000).Draw(t, "position")
			input = fmt.Sprintf("%d", pos)
			expectedPos = pos
			expectedIsTag = false
			expectedOptional = false

		case 1:
			// Non-negative integer position with ",optional" modifier: "N,optional"
			pos := rapid.IntRange(0, 10000).Draw(t, "position")
			input = fmt.Sprintf("%d,optional", pos)
			expectedPos = pos
			expectedIsTag = false
			expectedOptional = true

		case 2:
			// Dash with ",tag" modifier: "-,tag"
			input = "-,tag"
			expectedPos = -1
			expectedIsTag = true
			expectedOptional = false

		case 3:
			// Dash alone (skip field): "-"
			input = "-"
			expectedPos = -1
			expectedIsTag = false
			expectedOptional = false

		case 4:
			// Empty string (no wire tag, skip field)
			input = ""
			expectedPos = -1
			expectedIsTag = false
			expectedOptional = false
		}

		pos, isTag, optional, err := parseWireTag(input)
		if err != nil {
			t.Fatalf("parseWireTag(%q) returned unexpected error: %v", input, err)
		}
		if pos != expectedPos {
			t.Fatalf("parseWireTag(%q): expected position=%d, got %d", input, expectedPos, pos)
		}
		if isTag != expectedIsTag {
			t.Fatalf("parseWireTag(%q): expected isTag=%v, got %v", input, expectedIsTag, isTag)
		}
		if optional != expectedOptional {
			t.Fatalf("parseWireTag(%q): expected optional=%v, got %v", input, expectedOptional, optional)
		}
	})
}

// TestParseWireTag_InvalidStrings verifies that for any string NOT matching the
// wire tag grammar, parseWireTag returns an error.
// **Validates: Requirements 2.1, 2.2, 2.3, 2.4, 1.2, 1.3, 1.4**
func TestParseWireTag_InvalidStrings(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Decide which category of invalid wire tag to generate
		category := rapid.IntRange(0, 2).Draw(t, "category")

		var input string

		switch category {
		case 0:
			// Non-numeric, non-"-" position: random alphabetic string
			length := rapid.IntRange(1, 10).Draw(t, "length")
			chars := make([]byte, length)
			for i := range chars {
				chars[i] = byte(rapid.IntRange(int('a'), int('z')).Draw(t, fmt.Sprintf("char%d", i)))
			}
			input = string(chars)

		case 1:
			// Negative number (not just "-")
			negNum := rapid.IntRange(1, 10000).Draw(t, "negNum")
			input = fmt.Sprintf("-%d", negNum)

		case 2:
			// Valid position with unknown modifier
			pos := rapid.IntRange(0, 100).Draw(t, "position")
			// Generate a modifier that is not "tag" or "optional"
			modLength := rapid.IntRange(1, 8).Draw(t, "modLength")
			modChars := make([]byte, modLength)
			for i := range modChars {
				modChars[i] = byte(rapid.IntRange(int('a'), int('z')).Draw(t, fmt.Sprintf("modChar%d", i)))
			}
			modifier := string(modChars)
			// Ensure modifier is not one of the valid modifiers
			if modifier == "tag" || modifier == "optional" {
				modifier = "invalid"
			}
			input = fmt.Sprintf("%d,%s", pos, modifier)
		}

		_, _, _, err := parseWireTag(input)
		if err == nil {
			t.Fatalf("parseWireTag(%q) expected error but got nil", input)
		}
	})
}
