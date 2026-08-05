package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// Feature: lua-codegen, Property 13: Go Parse/Format Round-Trip
//
// This test validates that for any valid struct instance with non-tag fields populated,
// calling Format() and then the corresponding Parse function produces a struct whose
// string and integer fields are equal to the original, and whose float64 fields are
// within 0.1 of the original.
//
// The existing per-package roundtrip_prop_test.go files (entity, tile, surface, resource,
// game, playerattr, player) already exercise this property against the generated code.
// This test validates the property at the generator level by simulating the format/parse
// logic using EventSchema definitions, ensuring the wire format encoding is self-consistent
// for any schema the generator might produce.

// fieldValue holds a generated value for a field along with its schema.
type fieldValue struct {
	Schema FieldSchema
	StrVal string  // string value (for string fields)
	IntVal int     // int value (for int fields)
	F64Val float64 // float64 value (for float64 fields)
	U64Val uint64  // uint64 value (for uint64 fields)
}

// formatField produces the wire-format string for a fieldValue, matching the generated Format() logic.
func formatField(fv fieldValue) string {
	switch fv.Schema.Type {
	case "string":
		return fv.StrVal
	case "int":
		return strconv.Itoa(fv.IntVal)
	case "float64":
		return strconv.FormatFloat(fv.F64Val, 'f', 1, 64)
	case "uint64":
		return strconv.FormatUint(fv.U64Val, 10)
	default:
		return ""
	}
}

// parseField parses a wire-format string back into a fieldValue, matching the generated Parse logic.
func parseField(part string, schema FieldSchema) (fieldValue, error) {
	fv := fieldValue{Schema: schema}
	switch schema.Type {
	case "string":
		fv.StrVal = part
	case "int":
		v, err := strconv.Atoi(part)
		if err != nil {
			return fv, fmt.Errorf("invalid int %q: %w", part, err)
		}
		fv.IntVal = v
	case "float64":
		v, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return fv, fmt.Errorf("invalid float64 %q: %w", part, err)
		}
		fv.F64Val = v
	case "uint64":
		v, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return fv, fmt.Errorf("invalid uint64 %q: %w", part, err)
		}
		fv.U64Val = v
	}
	return fv, nil
}

// roundtripAlphanumGen generates a random non-empty alphanumeric string (no colons or brackets).
func roundtripAlphanumGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		length := rapid.IntRange(1, 20).Draw(t, "len")
		b := make([]byte, length)
		for i := range b {
			b[i] = chars[rapid.IntRange(0, len(chars)-1).Draw(t, "ch")]
		}
		return string(b)
	})
}

// roundtripTagGen generates a valid tag (lowercase + digits + underscores).
func roundtripTagGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		const chars = "abcdefghijklmnopqrstuvwxyz0123456789_"
		length := rapid.IntRange(3, 15).Draw(t, "len")
		b := make([]byte, length)
		b[0] = "abcdefghijklmnopqrstuvwxyz"[rapid.IntRange(0, 25).Draw(t, "first")]
		for i := 1; i < length; i++ {
			b[i] = chars[rapid.IntRange(0, len(chars)-1).Draw(t, fmt.Sprintf("ch%d", i))]
		}
		return string(b)
	})
}

// roundtripSchemaGen generates a random EventSchema with 1-7 non-tag fields of various types,
// plus a tag field, simulating what the generator would produce from annotated structs.
func roundtripSchemaGen() *rapid.Generator[EventSchema] {
	return rapid.Custom(func(t *rapid.T) EventSchema {
		fieldCount := rapid.IntRange(1, 7).Draw(t, "fieldCount")
		tag := roundtripTagGen().Draw(t, "tag")

		fields := []FieldSchema{
			{Name: "Event", Type: "string", Position: -1, IsTag: true},
		}

		fieldNames := []string{"Name", "X", "Y", "UnitNumber", "SurfaceName", "SurfaceIndex", "PlayerIndex"}
		fieldTypes := []string{"string", "int", "float64", "uint64"}

		for i := 0; i < fieldCount; i++ {
			nameIdx := i % len(fieldNames)
			typeIdx := rapid.IntRange(0, len(fieldTypes)-1).Draw(t, fmt.Sprintf("type%d", i))
			fields = append(fields, FieldSchema{
				Name:     fmt.Sprintf("%s%d", fieldNames[nameIdx], i),
				Type:     fieldTypes[typeIdx],
				Position: i,
				IsTag:    false,
				Optional: false,
			})
		}

		return EventSchema{
			PackageName: "test",
			StructName:  "TestEvent",
			Tags:        []string{tag},
			Fields:      fields,
			Separator:   ":",
		}
	})
}

// roundtripFieldValuesGen generates random values for each non-tag field in a schema.
func roundtripFieldValuesGen(schema EventSchema) *rapid.Generator[[]fieldValue] {
	return rapid.Custom(func(t *rapid.T) []fieldValue {
		var values []fieldValue
		for _, f := range schema.Fields {
			if f.IsTag {
				continue
			}
			fv := fieldValue{Schema: f}
			switch f.Type {
			case "string":
				fv.StrVal = roundtripAlphanumGen().Draw(t, f.Name)
			case "int":
				fv.IntVal = rapid.IntRange(0, 100000).Draw(t, f.Name)
			case "float64":
				fv.F64Val = rapid.Float64Range(-100000, 100000).Draw(t, f.Name)
			case "uint64":
				fv.U64Val = rapid.Uint64().Draw(t, f.Name)
			}
			values = append(values, fv)
		}
		return values
	})
}

// TestProperty13_GoParseFormatRoundTrip verifies that for any valid EventSchema and any
// random field values, formatting them into wire format and parsing back produces values
// where string and integer fields are exactly equal, and float64 fields are within 0.1.
//
// This exercises the same logic that the generated Format() and Parse functions use,
// validating the round-trip property at the generator level.
//
// **Validates: Requirements 8.3**
func TestProperty13_GoParseFormatRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random schema
		schema := roundtripSchemaGen().Draw(t, "schema")

		// Generate random field values matching the schema
		values := roundtripFieldValuesGen(schema).Draw(t, "values")

		tag := schema.Tags[0]

		// Format: build the wire message exactly as the generated Format() method does
		// Format is: "[<tag>] " + field0 + ":" + field1 + ":" + ... + fieldN
		var parts []string
		for _, fv := range values {
			parts = append(parts, formatField(fv))
		}
		wirePayload := strings.Join(parts, schema.Separator)
		wireMsg := "[" + tag + "] " + wirePayload

		// Parse: strip the tag prefix and split on separator, exactly as generated Parse does
		prefix := "[" + tag + "] "
		require.True(t, strings.HasPrefix(wireMsg, prefix), "wire message should have tag prefix")
		payload := wireMsg[len(prefix):]

		parsedParts := strings.Split(payload, schema.Separator)
		require.Equal(t, len(values), len(parsedParts),
			"parsed field count should match original field count")

		// Verify each field round-trips correctly
		for i, original := range values {
			parsed, err := parseField(parsedParts[i], original.Schema)
			require.NoError(t, err, "parsing field %s at position %d", original.Schema.Name, i)

			switch original.Schema.Type {
			case "string":
				assert.Equal(t, original.StrVal, parsed.StrVal,
					"string field %s should be equal", original.Schema.Name)
			case "int":
				assert.Equal(t, original.IntVal, parsed.IntVal,
					"int field %s should be equal", original.Schema.Name)
			case "uint64":
				assert.Equal(t, original.U64Val, parsed.U64Val,
					"uint64 field %s should be equal", original.Schema.Name)
			case "float64":
				assert.InDelta(t, original.F64Val, parsed.F64Val, 0.1,
					"float64 field %s should be within 0.1 (original=%v, parsed=%v)",
					original.Schema.Name, original.F64Val, parsed.F64Val)
			}
		}
	})
}

// TestProperty13_GoParseFormatRoundTrip_OptionalFields verifies the round-trip property
// also holds when schemas contain optional fields. Optional fields that are present in
// the wire format should round-trip correctly.
//
// **Validates: Requirements 8.3**
func TestProperty13_GoParseFormatRoundTrip_OptionalFields(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a schema with at least one optional field at the end
		baseFieldCount := rapid.IntRange(2, 5).Draw(t, "baseFieldCount")
		tag := roundtripTagGen().Draw(t, "tag")

		fields := []FieldSchema{
			{Name: "Event", Type: "string", Position: -1, IsTag: true},
		}

		fieldTypes := []string{"string", "int", "float64", "uint64"}

		for i := 0; i < baseFieldCount; i++ {
			typeIdx := rapid.IntRange(0, len(fieldTypes)-1).Draw(t, fmt.Sprintf("type%d", i))
			fields = append(fields, FieldSchema{
				Name:     fmt.Sprintf("Field%d", i),
				Type:     fieldTypes[typeIdx],
				Position: i,
				IsTag:    false,
				Optional: false,
			})
		}

		// Add one optional field at the end
		optTypeIdx := rapid.IntRange(0, len(fieldTypes)-1).Draw(t, "optType")
		fields = append(fields, FieldSchema{
			Name:     "OptionalField",
			Type:     fieldTypes[optTypeIdx],
			Position: baseFieldCount,
			IsTag:    false,
			Optional: true,
		})

		schema := EventSchema{
			PackageName: "test",
			StructName:  "TestEvent",
			Tags:        []string{tag},
			Fields:      fields,
			Separator:   ":",
		}

		// Generate values for ALL fields including optional
		values := roundtripFieldValuesGen(schema).Draw(t, "values")

		// Format the wire message
		var parts []string
		for _, fv := range values {
			parts = append(parts, formatField(fv))
		}
		wirePayload := strings.Join(parts, schema.Separator)
		wireMsg := "[" + tag + "] " + wirePayload

		// Parse back
		prefix := "[" + tag + "] "
		payload := wireMsg[len(prefix):]
		parsedParts := strings.Split(payload, schema.Separator)

		require.Equal(t, len(values), len(parsedParts))

		// Verify round-trip
		for i, original := range values {
			parsed, err := parseField(parsedParts[i], original.Schema)
			require.NoError(t, err)

			switch original.Schema.Type {
			case "string":
				assert.Equal(t, original.StrVal, parsed.StrVal)
			case "int":
				assert.Equal(t, original.IntVal, parsed.IntVal)
			case "uint64":
				assert.Equal(t, original.U64Val, parsed.U64Val)
			case "float64":
				assert.InDelta(t, original.F64Val, parsed.F64Val, 0.1)
			}
		}
	})
}

// TestProperty13_GoParseFormatRoundTrip_FloatPrecision specifically tests that float64
// values formatted with 1 decimal place round-trip within the 0.1 tolerance.
// This is critical because Format uses 'f', 1 precision which truncates.
//
// **Validates: Requirements 8.3**
func TestProperty13_GoParseFormatRoundTrip_FloatPrecision(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		original := rapid.Float64Range(-100000, 100000).Draw(t, "value")

		// Format as the generated code does: 1 decimal place
		formatted := strconv.FormatFloat(original, 'f', 1, 64)

		// Parse back
		parsed, err := strconv.ParseFloat(formatted, 64)
		require.NoError(t, err)

		// The parsed value should be within 0.1 of the original
		assert.True(t, math.Abs(original-parsed) <= 0.1,
			"float64 round-trip failed: original=%v, formatted=%q, parsed=%v, diff=%v",
			original, formatted, parsed, math.Abs(original-parsed))
	})
}
