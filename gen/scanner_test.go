package main

import (
	"go/ast"
	"go/token"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWireTag(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		pos      int
		isTag    bool
		optional bool
		wantErr  bool
	}{
		{name: "simple position", input: "0", pos: 0, isTag: false, optional: false},
		{name: "higher position", input: "5", pos: 5, isTag: false, optional: false},
		{name: "tag field", input: "-,tag", pos: -1, isTag: true, optional: false},
		{name: "skip field", input: "-", pos: -1, isTag: false, optional: false},
		{name: "optional field", input: "7,optional", pos: 7, isTag: false, optional: true},
		{name: "empty string", input: "", pos: -1, isTag: false, optional: false},
		{name: "invalid position", input: "abc", wantErr: true},
		{name: "negative position", input: "-1", wantErr: true},
		{name: "unknown modifier", input: "3,unknown", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pos, isTag, optional, err := parseWireTag(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.pos, pos)
			assert.Equal(t, tc.isTag, isTag)
			assert.Equal(t, tc.optional, optional)
		})
	}
}

func TestExtractAnnotations(t *testing.T) {
	cases := []struct {
		name     string
		comments []*ast.Comment
		want     []string
	}{
		{
			name: "single tag",
			comments: []*ast.Comment{
				{Text: "// gen:event tag=move"},
			},
			want: []string{"move"},
		},
		{
			name: "multiple tags",
			comments: []*ast.Comment{
				{Text: "// gen:event tag=entity_died"},
				{Text: "// gen:event tag=entity_built"},
				{Text: "// gen:event tag=entity_mined"},
			},
			want: []string{"entity_died", "entity_built", "entity_mined"},
		},
		{
			name: "no gen:event comments",
			comments: []*ast.Comment{
				{Text: "// This is a regular comment"},
			},
			want: nil,
		},
		{
			name: "mixed comments",
			comments: []*ast.Comment{
				{Text: "// Package doc comment"},
				{Text: "// gen:event tag=player_joined"},
				{Text: "// Another comment"},
			},
			want: []string{"player_joined"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := &ast.CommentGroup{List: tc.comments}
			got := extractAnnotations(doc)
			assert.Equal(t, tc.want, got)
		})
	}

	t.Run("nil comment group", func(t *testing.T) {
		got := extractAnnotations(nil)
		assert.Nil(t, got)
	})
}

func TestExtractAnnotations_Position(t *testing.T) {
	// Verify comments with various positions are handled.
	comments := []*ast.Comment{
		{Slash: token.Pos(10), Text: "// gen:event tag=test_event"},
	}
	doc := &ast.CommentGroup{List: comments}
	got := extractAnnotations(doc)
	assert.Equal(t, []string{"test_event"}, got)
}

func TestScanPackages_PlayerPackage(t *testing.T) {
	// Use the actual client directory to test scanning.
	dir := filepath.Join("..", "client")
	schemas, err := ScanPackages(dir)
	require.NoError(t, err)

	// Should find at least the Move struct in client/player.
	var moveSchema *EventSchema
	for i := range schemas {
		if schemas[i].StructName == "Move" && schemas[i].PackageName == "player" {
			moveSchema = &schemas[i]
			break
		}
	}

	require.NotNil(t, moveSchema, "expected to find Move schema in client/player")
	assert.Equal(t, "player", moveSchema.PackageName)
	assert.Equal(t, []string{"move"}, moveSchema.Tags)
	assert.Equal(t, ":", moveSchema.Separator)
	assert.Len(t, moveSchema.Fields, 6)

	// Verify field details.
	expectedFields := []FieldSchema{
		{Name: "Name", Type: "string", Position: 0, IsTag: false, Optional: false},
		{Name: "Index", Type: "int", Position: 1, IsTag: false, Optional: false},
		{Name: "X", Type: "float64", Position: 2, IsTag: false, Optional: false},
		{Name: "Y", Type: "float64", Position: 3, IsTag: false, Optional: false},
		{Name: "SurfaceName", Type: "string", Position: 4, IsTag: false, Optional: false},
		{Name: "SurfaceIndex", Type: "int", Position: 5, IsTag: false, Optional: false},
	}
	assert.Equal(t, expectedFields, moveSchema.Fields)
}

func TestComputePackagePath(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"/home/user/project/client/player", "client/player"},
		{"/home/user/project/client/entity", "client/entity"},
		{"client/player", "client/player"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := computePackagePath(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}
