package main

import (
	"fmt"
	"testing"

	"pgregory.net/rapid"
)

// Feature: event-schema-codegen, Property 5: Position Contiguity Validation

// supportedTypesList is the set of valid field types for generating test schemas.
var supportedTypesList = []string{"string", "int", "float64", "uint64"}

// TestProperty5_ContiguousPositionsAccepted verifies that any EventSchema with
// positions forming a contiguous sequence 0..N passes validation.
//
// **Validates: Requirements 2.6**
func TestProperty5_ContiguousPositionsAccepted(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a contiguous position count from 1 to 20.
		n := rapid.IntRange(1, 20).Draw(t, "fieldCount")

		fields := make([]FieldSchema, n)
		for i := 0; i < n; i++ {
			typeIdx := rapid.IntRange(0, len(supportedTypesList)-1).Draw(t, fmt.Sprintf("typeIdx_%d", i))
			fields[i] = FieldSchema{
				Name:     fmt.Sprintf("Field%d", i),
				Type:     supportedTypesList[typeIdx],
				Position: i,
				IsTag:    false,
			}
		}

		schema := EventSchema{
			StructName: "TestStruct",
			Tags:       []string{"test"},
			Fields:     fields,
		}

		err := Validate(schema)
		if err != nil {
			t.Fatalf("expected contiguous positions 0..%d to be valid, got error: %v", n-1, err)
		}
	})
}

// TestProperty5_NonContiguousPositionsRejected verifies that any EventSchema with
// positions that have gaps or do not start from 0 fails validation.
//
// **Validates: Requirements 2.6**
func TestProperty5_NonContiguousPositionsRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Strategy: generate a non-contiguous position set.
		// Either: positions don't start at 0, or there's a gap in the sequence.
		strategy := rapid.IntRange(0, 1).Draw(t, "strategy")

		var fields []FieldSchema

		switch strategy {
		case 0:
			// Positions don't start at 0: start at some offset > 0.
			offset := rapid.IntRange(1, 10).Draw(t, "offset")
			n := rapid.IntRange(1, 10).Draw(t, "fieldCount")
			fields = make([]FieldSchema, n)
			for i := 0; i < n; i++ {
				typeIdx := rapid.IntRange(0, len(supportedTypesList)-1).Draw(t, fmt.Sprintf("typeIdx_%d", i))
				fields[i] = FieldSchema{
					Name:     fmt.Sprintf("Field%d", i),
					Type:     supportedTypesList[typeIdx],
					Position: offset + i,
					IsTag:    false,
				}
			}

		case 1:
			// Gap in positions: contiguous from 0 but with one position removed.
			// Generate at least 2 positions so we can remove one and create a gap.
			n := rapid.IntRange(3, 12).Draw(t, "totalPositions")
			// Pick which position to remove (not the last one, to ensure a gap).
			removeIdx := rapid.IntRange(0, n-2).Draw(t, "removeIdx")

			fields = make([]FieldSchema, 0, n-1)
			for i := 0; i < n; i++ {
				if i == removeIdx {
					continue
				}
				typeIdx := rapid.IntRange(0, len(supportedTypesList)-1).Draw(t, fmt.Sprintf("typeIdx_gap_%d", i))
				fields = append(fields, FieldSchema{
					Name:     fmt.Sprintf("Field%d", i),
					Type:     supportedTypesList[typeIdx],
					Position: i,
					IsTag:    false,
				})
			}
		}

		schema := EventSchema{
			StructName: "TestStruct",
			Tags:       []string{"test"},
			Fields:     fields,
		}

		err := Validate(schema)
		if err == nil {
			var positions []int
			for _, f := range fields {
				positions = append(positions, f.Position)
			}
			t.Fatalf("expected non-contiguous positions %v to be rejected, but validation passed", positions)
		}
	})
}
