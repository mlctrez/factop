package main

import (
	"strings"
	"testing"
)

func TestValidate_ValidSchema(t *testing.T) {
	schema := EventSchema{
		StructName: "Move",
		Tags:       []string{"move"},
		Fields: []FieldSchema{
			{Name: "Name", Type: "string", Position: 0},
			{Name: "Index", Type: "int", Position: 1},
			{Name: "X", Type: "float64", Position: 2},
			{Name: "Y", Type: "float64", Position: 3},
			{Name: "SurfaceName", Type: "string", Position: 4},
			{Name: "SurfaceIndex", Type: "int", Position: 5},
		},
	}
	if err := Validate(schema); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidate_ValidSchemaWithTagField(t *testing.T) {
	schema := EventSchema{
		StructName: "EntityEvent",
		Tags:       []string{"entity_died", "entity_built"},
		Fields: []FieldSchema{
			{Name: "Event", Type: "string", Position: -1, IsTag: true},
			{Name: "Name", Type: "string", Position: 0},
			{Name: "X", Type: "float64", Position: 1},
			{Name: "Y", Type: "float64", Position: 2},
		},
	}
	if err := Validate(schema); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidate_NoEventTags(t *testing.T) {
	schema := EventSchema{
		StructName: "Move",
		Tags:       nil,
		Fields: []FieldSchema{
			{Name: "Name", Type: "string", Position: 0},
		},
	}
	err := Validate(schema)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no event tags defined") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "struct Move") {
		t.Fatalf("error should identify struct name: %v", err)
	}
}

func TestValidate_NoNonTagFields(t *testing.T) {
	schema := EventSchema{
		StructName: "Move",
		Tags:       []string{"move"},
		Fields: []FieldSchema{
			{Name: "Event", Type: "string", Position: -1, IsTag: true},
		},
	}
	err := Validate(schema)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no non-tag fields found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_DuplicatePosition(t *testing.T) {
	schema := EventSchema{
		StructName: "Move",
		Tags:       []string{"move"},
		Fields: []FieldSchema{
			{Name: "Name", Type: "string", Position: 0},
			{Name: "Index", Type: "int", Position: 1},
			{Name: "X", Type: "float64", Position: 1},
		},
	}
	err := Validate(schema)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate wire position 1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_NonContiguousPositions(t *testing.T) {
	schema := EventSchema{
		StructName: "Move",
		Tags:       []string{"move"},
		Fields: []FieldSchema{
			{Name: "Name", Type: "string", Position: 0},
			{Name: "Index", Type: "int", Position: 1},
			{Name: "X", Type: "float64", Position: 3},
		},
	}
	err := Validate(schema)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "missing position 2") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_UnsupportedType(t *testing.T) {
	schema := EventSchema{
		StructName: "Move",
		Tags:       []string{"move"},
		Fields: []FieldSchema{
			{Name: "Foo", Type: "complex128", Position: 0},
		},
	}
	err := Validate(schema)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported field type complex128 on field Foo") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_PositionsNotStartingAtZero(t *testing.T) {
	schema := EventSchema{
		StructName: "Move",
		Tags:       []string{"move"},
		Fields: []FieldSchema{
			{Name: "Name", Type: "string", Position: 1},
			{Name: "Index", Type: "int", Position: 2},
		},
	}
	err := Validate(schema)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "missing position 0") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_SkippedFieldsIgnored(t *testing.T) {
	// Fields with position -1 that aren't tags should be ignored.
	schema := EventSchema{
		StructName: "Move",
		Tags:       []string{"move"},
		Fields: []FieldSchema{
			{Name: "Internal", Type: "string", Position: -1, IsTag: false},
			{Name: "Name", Type: "string", Position: 0},
			{Name: "Index", Type: "int", Position: 1},
		},
	}
	if err := Validate(schema); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}
