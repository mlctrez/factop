package main

import (
	"fmt"
	"go/ast"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// Feature: lua-codegen, Property 1: Annotation Parsing Round-Trip

// luaEventNameGen generates a valid Factorio event name (e.g., "on_entity_died").
func luaEventNameGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		prefixes := []string{"on_entity_died", "on_built_entity", "on_robot_built_entity",
			"on_player_mined_entity", "on_robot_mined_entity", "on_player_built_tile",
			"on_robot_built_tile", "on_player_mined_tile", "on_robot_mined_tile",
			"on_surface_created", "on_surface_deleted", "on_resource_depleted",
			"on_player_joined_game", "on_player_left_game", "on_player_died",
			"on_player_respawned"}
		idx := rapid.IntRange(0, len(prefixes)-1).Draw(t, "eventIdx")
		return prefixes[idx]
	})
}

// luaFieldNameGen generates a valid Go field name (PascalCase, alphanumeric).
func luaFieldNameGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		names := []string{"Name", "X", "Y", "UnitNumber", "SurfaceName",
			"SurfaceIndex", "PlayerIndex", "Cause", "Amount", "Tick",
			"Color", "Health", "Force", "Type"}
		idx := rapid.IntRange(0, len(names)-1).Draw(t, "fieldNameIdx")
		return names[idx]
	})
}

// luaSourcePathGen generates a valid Lua source path expression.
func luaSourcePathGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		paths := []string{
			"event.entity.name", "event.entity.position.x", "event.entity.position.y",
			"event.entity.unit_number", "event.entity.surface.name", "event.entity.surface.index",
			"event.player_index", "event.surface.name", "event.surface.index",
			"game.tick", "tile.name", "tile.position.x", "tile.position.y",
			"event.entity.health", "event.entity.force.name",
		}
		idx := rapid.IntRange(0, len(paths)-1).Draw(t, "pathIdx")
		return paths[idx]
	})
}

// luaGuardGen generates a valid guard variable path.
func luaGuardGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		guards := []string{"event.entity", "event.entity.surface", "event.surface",
			"event.cause", "event.player"}
		idx := rapid.IntRange(0, len(guards)-1).Draw(t, "guardIdx")
		return guards[idx]
	})
}

// luaTagGen generates a valid tag string (lowercase + digits + underscores).
func luaTagGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		const chars = "abcdefghijklmnopqrstuvwxyz0123456789_"
		// Must be non-empty, start with a letter
		length := rapid.IntRange(1, 20).Draw(t, "tagLen")
		b := make([]byte, length)
		// First char: letter
		b[0] = "abcdefghijklmnopqrstuvwxyz"[rapid.IntRange(0, 25).Draw(t, "tagFirst")]
		for i := 1; i < length; i++ {
			b[i] = chars[rapid.IntRange(0, len(chars)-1).Draw(t, fmt.Sprintf("tagCh%d", i))]
		}
		return string(b)
	})
}

// luaDefaultValueGen generates a valid default literal value.
func luaDefaultValueGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		values := []string{"0", `""`, "0.0", "nil", "false"}
		idx := rapid.IntRange(0, len(values)-1).Draw(t, "defaultValIdx")
		return values[idx]
	})
}

// annotationInput holds the generated input for building annotation lines.
type annotationInput struct {
	Events   []string          // event names
	Fields   map[string]string // GoFieldName → source path
	Guards   []string          // guard variables
	Tag      string            // emitter tag
	Defaults map[string]string // GoFieldName → default value
	Iterate  string            // iteration source (may be empty)
	LoopVar  string            // loop variable (may be empty)
	NthTick  int               // nth_tick interval (0 = disabled)
}

// buildAnnotationComments constructs // gen:lua comment lines from an annotationInput.
func buildAnnotationComments(input annotationInput) *ast.CommentGroup {
	var comments []*ast.Comment

	// Event lines
	for _, ev := range input.Events {
		comments = append(comments, &ast.Comment{
			Text: fmt.Sprintf("// gen:lua event=%s", ev),
		})
	}

	// Tag line
	if input.Tag != "" {
		comments = append(comments, &ast.Comment{
			Text: fmt.Sprintf("// gen:lua tag=%s", input.Tag),
		})
	}

	// Guard lines
	for _, g := range input.Guards {
		comments = append(comments, &ast.Comment{
			Text: fmt.Sprintf("// gen:lua guard=%s", g),
		})
	}

	// Field lines
	for name, path := range input.Fields {
		comments = append(comments, &ast.Comment{
			Text: fmt.Sprintf("// gen:lua field.%s=%s", name, path),
		})
	}

	// Default lines
	for name, val := range input.Defaults {
		comments = append(comments, &ast.Comment{
			Text: fmt.Sprintf("// gen:lua default.%s=%s", name, val),
		})
	}

	// Iterate line
	if input.Iterate != "" {
		comments = append(comments, &ast.Comment{
			Text: fmt.Sprintf("// gen:lua iterate=%s", input.Iterate),
		})
	}

	// LoopVar line
	if input.LoopVar != "" {
		comments = append(comments, &ast.Comment{
			Text: fmt.Sprintf("// gen:lua loop_var=%s", input.LoopVar),
		})
	}

	// NthTick line
	if input.NthTick > 0 {
		comments = append(comments, &ast.Comment{
			Text: fmt.Sprintf("// gen:lua nth_tick=%d", input.NthTick),
		})
	}

	return &ast.CommentGroup{List: comments}
}

// annotationInputGen generates a random valid annotationInput.
func annotationInputGen() *rapid.Generator[annotationInput] {
	return rapid.Custom(func(t *rapid.T) annotationInput {
		input := annotationInput{
			Fields:   make(map[string]string),
			Defaults: make(map[string]string),
		}

		// Generate 1-5 events
		eventCount := rapid.IntRange(1, 5).Draw(t, "eventCount")
		seen := make(map[string]bool)
		for i := 0; i < eventCount; i++ {
			ev := luaEventNameGen().Draw(t, fmt.Sprintf("event_%d", i))
			if !seen[ev] {
				input.Events = append(input.Events, ev)
				seen[ev] = true
			}
		}

		// Generate tag
		input.Tag = luaTagGen().Draw(t, "tag")

		// Generate 0-3 guards
		guardCount := rapid.IntRange(0, 3).Draw(t, "guardCount")
		guardSeen := make(map[string]bool)
		for i := 0; i < guardCount; i++ {
			g := luaGuardGen().Draw(t, fmt.Sprintf("guard_%d", i))
			if !guardSeen[g] {
				input.Guards = append(input.Guards, g)
				guardSeen[g] = true
			}
		}

		// Generate 1-6 fields
		fieldCount := rapid.IntRange(1, 6).Draw(t, "fieldCount")
		for i := 0; i < fieldCount; i++ {
			name := luaFieldNameGen().Draw(t, fmt.Sprintf("fieldName_%d", i))
			path := luaSourcePathGen().Draw(t, fmt.Sprintf("fieldPath_%d", i))
			input.Fields[name] = path
		}

		// Generate 0-3 defaults (subset of fields)
		defaultCount := rapid.IntRange(0, 3).Draw(t, "defaultCount")
		fieldNames := make([]string, 0, len(input.Fields))
		for name := range input.Fields {
			fieldNames = append(fieldNames, name)
		}
		for i := 0; i < defaultCount && i < len(fieldNames); i++ {
			val := luaDefaultValueGen().Draw(t, fmt.Sprintf("defaultVal_%d", i))
			input.Defaults[fieldNames[i]] = val
		}

		// Optionally generate iterate + loop_var
		hasIterate := rapid.Bool().Draw(t, "hasIterate")
		if hasIterate {
			iterateSources := []string{"event.tiles", "event.items", "event.entities"}
			idx := rapid.IntRange(0, len(iterateSources)-1).Draw(t, "iterateIdx")
			input.Iterate = iterateSources[idx]

			loopVars := []string{"tile", "item", "entity"}
			input.LoopVar = loopVars[idx]
		}

		// Optionally generate nth_tick
		hasNthTick := rapid.Bool().Draw(t, "hasNthTick")
		if hasNthTick {
			input.NthTick = rapid.IntRange(1, 3600).Draw(t, "nthTick")
		}

		return input
	})
}

// TestLuaProperty1_AnnotationParsingRoundTrip verifies that for any valid set of
// gen:lua annotation lines, parsing them into a LuaSchema produces fields that match
// the generated input values (same keys and values, order-independent).
//
// **Validates: Requirements 1.1, 1.3, 1.4**
func TestLuaProperty1_AnnotationParsingRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random valid annotation input
		input := annotationInputGen().Draw(t, "input")

		// Build comment lines from the input
		commentGroup := buildAnnotationComments(input)

		// Parse the comments using extractLuaAnnotations
		schema := extractLuaAnnotations(commentGroup)

		// Verify schema is non-nil (we always generate at least one annotation)
		require.NotNil(t, schema, "expected non-nil LuaSchema from valid annotations")

		// Verify events match
		require.Len(t, schema.Events, len(input.Events), "event count mismatch")
		for i, ev := range input.Events {
			assert.Equal(t, ev, schema.Events[i].Name,
				"event name mismatch at index %d", i)
		}

		// Verify tag matches
		assert.Equal(t, input.Tag, schema.Tag, "tag mismatch")

		// Verify guards match (order preserved)
		require.Len(t, schema.Guards, len(input.Guards), "guard count mismatch")
		for i, g := range input.Guards {
			assert.Equal(t, g, schema.Guards[i], "guard mismatch at index %d", i)
		}

		// Verify fields match
		require.Len(t, schema.Fields, len(input.Fields), "field count mismatch")
		for name, path := range input.Fields {
			schemaPath, ok := schema.Fields[name]
			require.True(t, ok, "field %q not found in parsed schema", name)
			assert.Equal(t, path, schemaPath, "field %q path mismatch", name)
		}

		// Verify defaults match
		require.Len(t, schema.Defaults, len(input.Defaults), "defaults count mismatch")
		for name, val := range input.Defaults {
			schemaVal, ok := schema.Defaults[name]
			require.True(t, ok, "default %q not found in parsed schema", name)
			assert.Equal(t, val, schemaVal, "default %q value mismatch", name)
		}

		// Verify iterate
		assert.Equal(t, input.Iterate, schema.Iterate, "iterate mismatch")

		// Verify loop_var
		assert.Equal(t, input.LoopVar, schema.LoopVar, "loop_var mismatch")

		// Verify nth_tick
		if input.NthTick > 0 {
			assert.Equal(t, input.NthTick, schema.NthTick, "nth_tick mismatch")
		} else {
			assert.Equal(t, 0, schema.NthTick, "nth_tick should be 0 when not specified")
		}

		// Verify include_cause is false (we didn't generate it)
		assert.False(t, schema.IncludeCause, "include_cause should be false when not set")
	})
}

// TestLuaProperty1_AnnotationParsingRoundTrip_WithIncludeCause verifies that
// include_cause annotation is correctly parsed.
//
// **Validates: Requirements 1.1, 1.3, 1.4**
func TestLuaProperty1_AnnotationParsingRoundTrip_WithIncludeCause(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a tag for include_cause value
		causeTag := luaTagGen().Draw(t, "causeTag")

		comments := &ast.CommentGroup{
			List: []*ast.Comment{
				{Text: "// gen:lua event=on_entity_died"},
				{Text: "// gen:lua tag=entity_died"},
				{Text: fmt.Sprintf("// gen:lua include_cause=%s", causeTag)},
			},
		}

		schema := extractLuaAnnotations(comments)
		require.NotNil(t, schema)

		assert.True(t, schema.IncludeCause, "include_cause should be true when set")
		assert.Equal(t, "entity_died", schema.Tag)
		assert.Len(t, schema.Events, 1)
		assert.Equal(t, "on_entity_died", schema.Events[0].Name)
	})
}

// TestLuaProperty1_NilCommentGroup verifies that nil input returns nil schema.
//
// **Validates: Requirements 1.1, 1.3, 1.4**
func TestLuaProperty1_NilCommentGroup(t *testing.T) {
	schema := extractLuaAnnotations(nil)
	assert.Nil(t, schema, "nil comment group should produce nil schema")
}

// TestLuaProperty1_NoGenLuaComments verifies that comments without gen:lua prefix return nil.
//
// **Validates: Requirements 1.1, 1.3, 1.4**
func TestLuaProperty1_NoGenLuaComments(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random non-gen:lua comments
		count := rapid.IntRange(1, 5).Draw(t, "commentCount")
		var comments []*ast.Comment
		for i := 0; i < count; i++ {
			comments = append(comments, &ast.Comment{
				Text: fmt.Sprintf("// regular comment %d", i),
			})
		}

		schema := extractLuaAnnotations(&ast.CommentGroup{List: comments})
		assert.Nil(t, schema, "comments without gen:lua should produce nil schema")
	})
}

// TestLuaProperty1_NthTickParsing verifies that nth_tick values are correctly parsed
// as integers from annotation strings.
//
// **Validates: Requirements 1.1, 1.3, 1.4**
func TestLuaProperty1_NthTickParsing(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		interval := rapid.IntRange(1, 3600).Draw(t, "interval")

		comments := &ast.CommentGroup{
			List: []*ast.Comment{
				{Text: "// gen:lua event=on_entity_died"},
				{Text: "// gen:lua tag=test_tag"},
				{Text: fmt.Sprintf("// gen:lua nth_tick=%s", strconv.Itoa(interval))},
			},
		}

		schema := extractLuaAnnotations(comments)
		require.NotNil(t, schema)
		assert.Equal(t, interval, schema.NthTick, "nth_tick should match generated interval")
	})
}
