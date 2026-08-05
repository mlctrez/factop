package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// supportedTypes is the set of Go types allowed in wire-tagged fields.
var supportedTypes = map[string]bool{
	"string":  true,
	"int":     true,
	"float64": true,
	"uint64":  true,
}

// Validate checks an EventSchema for correctness.
// Returns an error describing the first issue found.
func Validate(schema EventSchema) error {
	name := schema.StructName

	// Must have at least one event tag defined.
	if len(schema.Tags) == 0 {
		return fmt.Errorf("struct %s: no event tags defined", name)
	}

	// Collect non-tag field positions and check types.
	var positions []int
	seen := make(map[int]bool)
	hasNonTag := false

	for _, f := range schema.Fields {
		if f.IsTag {
			continue
		}

		// Fields with position -1 that aren't tags are skipped fields.
		if f.Position == -1 {
			continue
		}

		hasNonTag = true

		// Check for supported types.
		if !supportedTypes[f.Type] {
			return fmt.Errorf("struct %s: unsupported field type %s on field %s", name, f.Type, f.Name)
		}

		// Check for duplicate positions.
		if seen[f.Position] {
			return fmt.Errorf("struct %s: duplicate wire position %d", name, f.Position)
		}
		seen[f.Position] = true
		positions = append(positions, f.Position)
	}

	// Must have at least one non-tag field.
	if !hasNonTag {
		return fmt.Errorf("struct %s: no non-tag fields found", name)
	}

	// Positions must form a contiguous sequence starting from 0.
	sort.Ints(positions)
	for i, pos := range positions {
		if pos != i {
			return fmt.Errorf("struct %s: positions are not contiguous from 0 (missing position %d)", name, i)
		}
	}

	return nil
}

// tagPattern defines valid tag format: lowercase letters, digits, and underscores.
var tagPattern = regexp.MustCompile(`^[a-z0-9_]+$`)

// recognizedKeyPrefixes are key prefixes that are always valid.
var recognizedKeyPrefixes = []string{"field.", "default.", "fallback."}

// recognizedKeys are exact key matches that are valid.
var recognizedKeys = map[string]bool{
	"event":         true,
	"guard":         true,
	"iterate":       true,
	"loop_var":      true,
	"tag":           true,
	"nth_tick":      true,
	"include_cause": true,
}

// ValidateLua checks Lua-specific schema annotations for correctness.
// Returns an error describing the first issue found.
func ValidateLua(schema EventSchema) error {
	lua := schema.Lua
	if lua == nil {
		return nil
	}
	name := schema.StructName

	// Validate tag is non-empty and matches pattern.
	if lua.Tag == "" {
		return fmt.Errorf("struct %s: missing required tag annotation", name)
	}
	if !tagPattern.MatchString(lua.Tag) {
		return fmt.Errorf("struct %s: invalid tag %q (must match ^[a-z0-9_]+$)", name, lua.Tag)
	}

	// Validate event names are non-empty strings.
	for _, ev := range lua.Events {
		if ev.Name == "" {
			return fmt.Errorf("struct %s: event name must be non-empty", name)
		}
	}

	// Validate no more than 8 event annotations per struct.
	if len(lua.Events) > 8 {
		return fmt.Errorf("struct %s: too many event annotations (%d, max 8)", name, len(lua.Events))
	}

	// Validate guard variables are non-empty strings.
	for _, g := range lua.Guards {
		if strings.TrimSpace(g) == "" {
			return fmt.Errorf("struct %s: guard variable must be non-empty", name)
		}
	}

	// Validate all fields in LuaSchema.Fields exist in EventSchema.Fields.
	eventFields := make(map[string]bool)
	for _, f := range schema.Fields {
		eventFields[f.Name] = true
	}
	for fieldName := range lua.Fields {
		if !eventFields[fieldName] {
			return fmt.Errorf("struct %s: lua field %q not found in struct fields", name, fieldName)
		}
	}

	// Validate NthTick is in range [1, 3600] when specified (> 0).
	if lua.NthTick != 0 {
		if lua.NthTick < 1 || lua.NthTick > 3600 {
			return fmt.Errorf("struct %s: nth_tick interval %d out of range [1, 3600]", name, lua.NthTick)
		}
	}

	// Validate Fallback covers all non-tag fields when specified.
	if lua.Fallback != nil {
		for _, f := range schema.Fields {
			if f.IsTag {
				continue
			}
			if f.Position == -1 {
				continue
			}
			if _, ok := lua.Fallback.Defaults[f.Name]; !ok {
				return fmt.Errorf("struct %s: fallback missing definition for field %q", name, f.Name)
			}
		}
	}

	// Detect unrecognized annotation keys.
	for _, key := range lua.RawKeys {
		if isRecognizedKey(key) {
			continue
		}
		return fmt.Errorf("struct %s: unrecognized gen:lua key %q", name, key)
	}

	return nil
}

// isRecognizedKey checks if a key is in the recognized set.
func isRecognizedKey(key string) bool {
	if recognizedKeys[key] {
		return true
	}
	for _, prefix := range recognizedKeyPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}
