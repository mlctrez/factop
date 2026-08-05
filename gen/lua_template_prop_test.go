package main

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Feature: lua-codegen, Property 8: Generated Module Structure Validity

// genPackageName generates a valid lowercase package name suitable for Lua module naming.
func genPackageName() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-z][a-z0-9]{1,10}`)
}

// genFieldType generates a valid Go field type used in EventSchema.
func genFieldType() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{"string", "int", "float64", "uint64"})
}

// genLuaSourcePath generates a plausible Lua source path expression.
func genLuaSourcePath() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{
		"event.entity.name",
		"event.entity.position.x",
		"event.entity.position.y",
		"event.entity.unit_number",
		"event.player_index",
		"entity.name",
		"entity.position.x",
		"entity.position.y",
		"surface.name",
		"surface.index",
		"game.tick",
		"p.name",
		"p.index",
		"tile.old_tile.name",
		"tile.position.x",
		"tile.position.y",
	})
}

// schemaPattern defines a generation strategy for building varied EventSchemas.
type schemaPattern int

const (
	patternSimple    schemaPattern = iota // single event, no iterate
	patternMulti                          // multiple events sharing fields
	patternNthTick                        // nth_tick registration
	patternIterate                        // iteration pattern
)

// genEventSchemaWithLua generates random EventSchemas with Lua annotations
// covering all four structural patterns: simple, multi-event, nth_tick, iterate.
func genEventSchemaWithLua() *rapid.Generator[EventSchema] {
	return rapid.Custom[EventSchema](func(t *rapid.T) EventSchema {
		pkg := genPackageName().Draw(t, "pkg")
		structName := rapid.StringMatching(`[A-Z][A-Za-z]{2,12}`).Draw(t, "structName")
		tag := rapid.StringMatching(`[a-z][a-z0-9_]{1,12}`).Draw(t, "tag")

		// Generate 1-5 non-tag fields.
		numFields := rapid.IntRange(1, 5).Draw(t, "numFields")
		fields := make([]FieldSchema, 0, numFields+1)

		// Optionally add a tag field.
		hasTagField := rapid.Bool().Draw(t, "hasTagField")
		if hasTagField {
			fields = append(fields, FieldSchema{
				Name:     "Event",
				Type:     "string",
				Position: -1,
				IsTag:    true,
			})
		}

		// Generate non-tag fields at positions 0..numFields-1.
		fieldNames := []string{"Name", "X", "Y", "Amount", "Index", "Value", "Surface", "Player", "Unit", "Tick"}
		usedNames := make(map[string]bool)
		for i := 0; i < numFields; i++ {
			// Pick a unique field name.
			var fname string
			for {
				idx := rapid.IntRange(0, len(fieldNames)-1).Draw(t, fmt.Sprintf("fnameIdx_%d", i))
				fname = fieldNames[idx]
				if !usedNames[fname] {
					usedNames[fname] = true
					break
				}
			}
			ftype := genFieldType().Draw(t, fmt.Sprintf("ftype_%d", i))
			fields = append(fields, FieldSchema{
				Name:     fname,
				Type:     ftype,
				Position: i,
			})
		}

		// Pick a pattern.
		pattern := schemaPattern(rapid.IntRange(0, 3).Draw(t, "pattern"))

		var luaSchema LuaSchema
		luaSchema.Tag = tag
		luaSchema.Fields = make(map[string]string)
		luaSchema.Defaults = make(map[string]string)

		// Build field mappings for all non-tag fields.
		for _, f := range fields {
			if f.IsTag {
				continue
			}
			luaSchema.Fields[f.Name] = genLuaSourcePath().Draw(t, fmt.Sprintf("sourcePath_%s", f.Name))
		}

		tags := []string{tag}

		switch pattern {
		case patternSimple:
			// Single event, simple handler.
			luaSchema.Events = []LuaEvent{{Name: "on_" + tag, Tag: tag}}

		case patternMulti:
			// 2-4 events sharing the same fields.
			numEvents := rapid.IntRange(2, 4).Draw(t, "numEvents")
			for i := 0; i < numEvents; i++ {
				evTag := tag
				if i > 0 {
					evTag = rapid.StringMatching(`[a-z][a-z0-9_]{1,10}`).Draw(t, fmt.Sprintf("evTag_%d", i))
					tags = append(tags, evTag)
				}
				luaSchema.Events = append(luaSchema.Events, LuaEvent{
					Name: fmt.Sprintf("on_%s_%d", tag, i),
					Tag:  evTag,
				})
			}

		case patternNthTick:
			// nth_tick pattern: no event parameter, game.tick source.
			interval := rapid.IntRange(1, 3600).Draw(t, "nthTick")
			luaSchema.NthTick = interval
			luaSchema.Events = []LuaEvent{{Name: "on_tick", Tag: tag}}
			// Override field sources to not use event.* paths.
			for _, f := range fields {
				if f.IsTag {
					continue
				}
				luaSchema.Fields[f.Name] = "game." + strings.ToLower(f.Name)
			}

		case patternIterate:
			// Iteration pattern.
			luaSchema.Iterate = "event.items"
			luaSchema.LoopVar = "item"
			luaSchema.Events = []LuaEvent{
				{Name: "on_" + tag + "_a", Tag: tag},
				{Name: "on_" + tag + "_b", Tag: tag},
			}
			// Make some fields loop-scoped and some event-scoped.
			for _, f := range fields {
				if f.IsTag {
					continue
				}
				if rapid.Bool().Draw(t, fmt.Sprintf("loopScoped_%s", f.Name)) {
					luaSchema.Fields[f.Name] = "item." + strings.ToLower(f.Name)
				} else {
					luaSchema.Fields[f.Name] = "event." + strings.ToLower(f.Name)
				}
			}
		}

		// Optionally add guards for non-nth_tick patterns.
		if pattern != patternNthTick {
			numGuards := rapid.IntRange(0, 2).Draw(t, "numGuards")
			for i := 0; i < numGuards; i++ {
				guard := rapid.SampledFrom([]string{
					"event.entity", "entity.surface",
					"game.surfaces[event.surface_index]",
					"game.players[event.player_index]",
				}).Draw(t, fmt.Sprintf("guard_%d", i))
				luaSchema.Guards = append(luaSchema.Guards, guard)
			}
		}

		return EventSchema{
			PackageName: pkg,
			PackagePath: "client/" + pkg,
			StructName:  structName,
			Tags:        tags,
			Fields:      fields,
			Separator:   ":",
			Lua:         &luaSchema,
		}
	})
}

// globalAssignmentPattern matches lines that assign to a global variable.
// A global assignment is a non-local, non-table, non-comment line with an "=" that
// is not a table field assignment (pkg_mod.field = ...) and not inside a string.
var globalAssignmentPattern = regexp.MustCompile(`^\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*`)

// TestProperty8_GeneratedModuleStructureValidity verifies that for any EventSchema
// with Lua annotations, the generated Lua file: (a) begins with the code-generated
// header, (b) declares a local <pkg>_mod = {} table, (c) contains only local scoped
// declarations (no global assignments), and (d) ends with return <pkg>_mod.
//
// **Validates: Requirements 2.2, 7.2, 7.3, 7.4**
func TestProperty8_GeneratedModuleStructureValidity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		schema := genEventSchemaWithLua().Draw(t, "schema")

		var buf bytes.Buffer
		err := luaTemplate.Execute(&buf, schema)
		if err != nil {
			t.Fatalf("template execution failed: %v", err)
		}

		output := buf.String()

		// (a) Verify output starts with the code-generated header.
		expectedHeader := "-- Code generated by go run ./gen -- DO NOT EDIT.\n"
		if !strings.HasPrefix(output, expectedHeader) {
			t.Fatalf("output does not start with expected header.\nGot prefix: %q", output[:min(len(output), 80)])
		}

		// (b) Verify output contains local <pkg>_mod = {} declaration.
		modDecl := fmt.Sprintf("local %s_mod = {}", schema.PackageName)
		if !strings.Contains(output, modDecl) {
			t.Fatalf("output does not contain module declaration %q", modDecl)
		}

		// (c) Verify no global assignments — all function/variable declarations use local.
		lines := strings.Split(output, "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)

			// Skip empty lines, comments, and return statements.
			if trimmed == "" || strings.HasPrefix(trimmed, "--") || strings.HasPrefix(trimmed, "return ") {
				continue
			}

			// Skip lines inside string.format (continuation args).
			if strings.HasPrefix(trimmed, "tag,") || strings.HasPrefix(trimmed, "\"") {
				continue
			}

			// Check function declarations: every "function " must be preceded by "local ".
			if strings.Contains(trimmed, "function ") && !strings.HasPrefix(trimmed, "local function ") {
				// Allow anonymous functions and method-style table assignments.
				// Not a standalone function declaration if it's an assignment to pkg_mod.
				if !strings.Contains(trimmed, schema.PackageName+"_mod") &&
					!strings.HasPrefix(trimmed, "if ") &&
					!strings.HasPrefix(trimmed, "elseif ") &&
					!strings.HasPrefix(trimmed, "end") {
					t.Fatalf("line %d has non-local function declaration: %q", i+1, trimmed)
				}
			}

			// Check variable assignments: non-comment lines with "=" that aren't:
			// - local declarations
			// - table field assignments (pkg_mod.X = ...)
			// - inside control flow (if/for/etc.)
			// - string content or format args
			if globalAssignmentPattern.MatchString(trimmed) {
				varName := globalAssignmentPattern.FindStringSubmatch(trimmed)[1]

				// Allow "local" prefix (the regex matches the var name after local too, but we check the full line).
				if strings.HasPrefix(trimmed, "local ") {
					continue
				}

				// Allow table field assignments on pkg_mod.
				if strings.HasPrefix(trimmed, schema.PackageName+"_mod.") {
					continue
				}

				// Allow assignments to loop variables or known patterns.
				if varName == "cause" || varName == "msg" {
					// These should be local — flag if not.
					t.Fatalf("line %d has potential global assignment to %q: %q", i+1, varName, trimmed)
				}

				// Allow lines that are table entries [x] = y.
				if strings.HasPrefix(trimmed, "[") {
					continue
				}

				t.Fatalf("line %d has global assignment to %q: %q", i+1, varName, trimmed)
			}
		}

		// (d) Verify output ends with "return <pkg>_mod\n".
		expectedReturn := fmt.Sprintf("return %s_mod\n", schema.PackageName)
		if !strings.HasSuffix(output, expectedReturn) {
			// Show the last few lines for debugging.
			lastLines := output
			if len(output) > 200 {
				lastLines = output[len(output)-200:]
			}
			t.Fatalf("output does not end with %q.\nLast chars: %q", expectedReturn, lastLines)
		}
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Feature: lua-codegen, Property 12: Fallback Branch Format Consistency

// fallbackFieldNames is a fixed set of distinct Go field names for fallback schema generation.
var fallbackFieldNames = []string{
	"Alpha", "Beta", "Gamma", "Delta", "Epsilon",
	"Zeta", "Eta", "Theta", "Iota", "Kappa",
}

// fallbackDefaultForType returns a Lua fallback default expression appropriate for a field type.
func fallbackDefaultForType(fieldType string) string {
	switch fieldType {
	case "string":
		return `""`
	case "int", "uint64":
		return "0"
	case "float64":
		return "0.0"
	default:
		return `""`
	}
}

// TestProperty12_FallbackBranchFormatConsistency verifies that for any EventSchema
// with a fallback annotation, both the valid-object branch and the fallback branch
// produce string.format calls with identical format strings (same separator count,
// same placeholder count, same tag prefix), differing only in the argument expressions.
//
// **Validates: Requirements 6.1, 6.3**
func TestProperty12_FallbackBranchFormatConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate 2-5 non-tag fields with sequential positions.
		numFields := rapid.IntRange(2, 5).Draw(t, "fieldCount")
		fields := make([]FieldSchema, 0, numFields+1)

		// Add a tag field.
		fields = append(fields, FieldSchema{
			Name:     "Event",
			Type:     "string",
			Position: -1,
			IsTag:    true,
		})

		// Generate non-tag fields with random types.
		luaFields := make(map[string]string)
		fallbackDefaults := make(map[string]string)
		for i := 0; i < numFields; i++ {
			fieldType := genFieldType().Draw(t, "fieldType")
			fieldName := fallbackFieldNames[i]
			fields = append(fields, FieldSchema{
				Name:     fieldName,
				Type:     fieldType,
				Position: i,
				IsTag:    false,
				Optional: false,
			})
			// Give each field a source path expression.
			luaFields[fieldName] = "surface." + strings.ToLower(fieldName)
			// Use a type-appropriate fallback default for each field.
			fallbackDefaults[fieldName] = fallbackDefaultForType(fieldType)
		}

		// Generate a fallback object name.
		fallbackObject := "surface"

		// Generate two tags: one for normal event, one for fallback event.
		normalTag := "surface_created"
		fallbackTag := "surface_deleted"

		schema := EventSchema{
			PackageName: "surface",
			PackagePath: "client/surface",
			StructName:  "SurfaceEvent",
			Tags:        []string{normalTag, fallbackTag},
			Fields:      fields,
			Separator:   ":",
			Lua: &LuaSchema{
				Events: []LuaEvent{
					{Name: "on_surface_created", Tag: normalTag},
					{Name: "on_surface_deleted", Tag: fallbackTag},
				},
				Fields:   luaFields,
				Guards:   []string{"game.surfaces[event.surface_index]"},
				Tag:      normalTag,
				Defaults: map[string]string{},
				Fallback: &LuaFallback{
					Object:   fallbackObject,
					Defaults: fallbackDefaults,
				},
			},
		}

		// Execute the template.
		var buf bytes.Buffer
		err := luaTemplate.Execute(&buf, schema)
		if err != nil {
			t.Fatalf("template execution failed: %v", err)
		}

		output := buf.String()

		// Find the fallback handler function (on_surface_deleted).
		// It should contain two string.format calls: one in the valid branch, one in else.
		handlerStart := strings.Index(output, "local function on_surface_deleted(event)")
		if handlerStart == -1 {
			t.Fatalf("could not find on_surface_deleted handler in output:\n%s", output)
		}

		// Find the next handler or end of file to scope our search.
		handlerSection := output[handlerStart:]
		endIdx := strings.Index(handlerSection, "\nend\n")
		if endIdx == -1 {
			// Try alternate end pattern.
			endIdx = strings.Index(handlerSection, "\nend")
			if endIdx == -1 {
				t.Fatalf("could not find end of on_surface_deleted handler:\n%s", handlerSection)
			}
		}
		handlerBody := handlerSection[:endIdx]

		// Extract all string.format calls from the handler body.
		formatCalls := extractFormatStrings(handlerBody)
		if len(formatCalls) < 2 {
			t.Fatalf("expected at least 2 string.format calls in fallback handler, got %d.\nHandler body:\n%s",
				len(formatCalls), handlerBody)
		}

		// The first format call is the valid branch, the second is the fallback branch.
		validFmt := formatCalls[0]
		fallbackFmt := formatCalls[1]

		// Property: both format strings must be identical.
		if validFmt != fallbackFmt {
			t.Fatalf("format strings differ between valid and fallback branches:\n  valid:    %q\n  fallback: %q\n\nHandler body:\n%s",
				validFmt, fallbackFmt, handlerBody)
		}

		// Additional verification: the format string should have the correct number of
		// colon-separated specifiers equal to the non-tag field count.
		// The format pattern is "[tag] spec1:spec2:...:specN"
		tagPrefix := "[" + fallbackTag + "] "
		if !strings.HasPrefix(validFmt, tagPrefix) {
			t.Fatalf("format string %q does not start with expected tag prefix %q", validFmt, tagPrefix)
		}
		specPart := strings.TrimPrefix(validFmt, tagPrefix)
		specifiers := strings.Split(specPart, ":")
		if len(specifiers) != numFields {
			t.Fatalf("expected %d specifiers in format string, got %d (format: %q)",
				numFields, len(specifiers), validFmt)
		}
	})
}

// extractFormatStrings finds all string.format("...") format string arguments in Lua code.
// It returns just the format string portion (the first argument to string.format).
func extractFormatStrings(code string) []string {
	var results []string
	searchFrom := 0
	for {
		idx := strings.Index(code[searchFrom:], "string.format(\"")
		if idx == -1 {
			break
		}
		// Move to the start of the format string (after the opening quote).
		fmtStart := searchFrom + idx + len("string.format(\"")
		// Find the closing quote of the format string.
		fmtEnd := strings.Index(code[fmtStart:], "\"")
		if fmtEnd == -1 {
			break
		}
		results = append(results, code[fmtStart:fmtStart+fmtEnd])
		searchFrom = fmtStart + fmtEnd + 1
	}
	return results
}

// Feature: lua-codegen, Property 7: Multi-Event Shared Emit Structure

// genMultiEventSchema generates an EventSchema with M > 1 Lua event annotations
// sharing the same field set, with IncludeCause=true to trigger shared emit pattern.
func genMultiEventSchema() *rapid.Generator[EventSchema] {
	return rapid.Custom[EventSchema](func(t *rapid.T) EventSchema {
		// Generate 2-5 events.
		eventCount := rapid.IntRange(2, 5).Draw(t, "eventCount")

		// Generate a package name.
		pkg := rapid.SampledFrom([]string{"entity", "widget", "thing", "item", "unit"}).Draw(t, "pkg")

		// Generate a tag for the schema.
		tag := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "tag")

		// Generate event names and assign tags.
		eventNames := []string{
			"on_entity_died", "on_built_entity", "on_robot_built_entity",
			"on_player_mined_entity", "on_robot_mined_entity",
			"on_entity_damaged", "on_entity_cloned", "on_entity_destroyed",
		}
		events := make([]LuaEvent, eventCount)
		for i := 0; i < eventCount; i++ {
			idx := rapid.IntRange(0, len(eventNames)-1).Draw(t, "eventNameIdx")
			events[i] = LuaEvent{
				Name: eventNames[idx],
				Tag:  tag,
			}
		}

		// Generate 1-5 non-tag fields.
		fieldCount := rapid.IntRange(1, 5).Draw(t, "fieldCount")
		fields := make([]FieldSchema, 0, fieldCount+1)
		// Add a tag field.
		fields = append(fields, FieldSchema{
			Name:     "Event",
			Type:     "string",
			Position: -1,
			IsTag:    true,
		})
		luaFields := make(map[string]string)
		for i := 0; i < fieldCount; i++ {
			fieldType := genFieldType().Draw(t, "fieldType")
			fieldName := fmt.Sprintf("Field%d", i)
			fields = append(fields, FieldSchema{
				Name:     fieldName,
				Type:     fieldType,
				Position: i,
				IsTag:    false,
			})
			luaFields[fieldName] = "event." + strings.ToLower(fieldName)
		}

		schema := EventSchema{
			PackageName: pkg,
			PackagePath: "client/" + pkg,
			StructName:  "TestStruct",
			Tags:        []string{tag},
			Fields:      fields,
			Lua: &LuaSchema{
				Events:       events,
				Fields:       luaFields,
				Guards:       []string{"event.entity"},
				Tag:          tag,
				IncludeCause: true,
				Defaults:     map[string]string{},
			},
		}

		return schema
	})
}

// TestProperty7_MultiEventSharedEmitStructure verifies that for any EventSchema
// with M > 1 Lua event annotations sharing the same field set, the generated Lua
// output contains exactly one shared local emit function and exactly M handler
// functions, where each handler's body consists solely of a call to the shared
// emit function.
//
// **Validates: Requirements 1.7, 3.1, 3.2**
func TestProperty7_MultiEventSharedEmitStructure(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		schema := genMultiEventSchema().Draw(t, "schema")

		// Render the Lua output.
		var buf bytes.Buffer
		err := luaTemplate.Execute(&buf, schema)
		if err != nil {
			t.Fatalf("template execution failed: %v", err)
		}
		output := buf.String()

		pkg := schema.PackageName
		emitFuncPrefix := "local function emit_" + pkg

		// Property: exactly one shared emit function.
		emitCount := strings.Count(output, emitFuncPrefix)
		if emitCount != 1 {
			t.Fatalf("expected exactly 1 occurrence of %q, got %d\nOutput:\n%s",
				emitFuncPrefix, emitCount, output)
		}

		// Property: exactly M handler functions (one per event).
		eventCount := len(schema.Lua.Events)
		handlerPrefix := "local function on_"
		handlerCount := strings.Count(output, handlerPrefix)
		if handlerCount != eventCount {
			t.Fatalf("expected %d handler functions (prefix %q), got %d\nOutput:\n%s",
				eventCount, handlerPrefix, handlerCount, output)
		}

		// Property: each handler calls the shared emit function.
		emitCallPrefix := "emit_" + pkg + "("
		for _, ev := range schema.Lua.Events {
			// Find the handler function body.
			handlerStart := strings.Index(output, "local function "+ev.Name+"(event)")
			if handlerStart == -1 {
				t.Fatalf("handler function %q not found in output:\n%s", ev.Name, output)
			}
			// Extract body from handler start to the next "end" or "local function".
			afterHandler := output[handlerStart:]
			lines := strings.SplitN(afterHandler, "\n", 10)
			foundEmitCall := false
			for _, line := range lines[1:] {
				if strings.Contains(line, emitCallPrefix) {
					foundEmitCall = true
					break
				}
				// Stop at end of function.
				if strings.TrimSpace(line) == "end" {
					break
				}
			}
			if !foundEmitCall {
				t.Fatalf("handler %q does not call %q\nHandler section:\n%s",
					ev.Name, emitCallPrefix, strings.Join(lines[:5], "\n"))
			}
		}
	})
}

// Feature: lua-codegen, Property 10: Optional Field Nil-Coalescing

// defaultForType returns the expected Lua nil-coalescing default for a given Go type.
func defaultForType(goType string) string {
	switch goType {
	case "int", "uint64":
		return "0"
	case "float64":
		return "0.0"
	case "string":
		return `""`
	default:
		return "0"
	}
}

// TestProperty10_OptionalFieldNilCoalescing verifies that for any EventSchema
// containing optional fields (wire modifier `optional`), the generated Lua code
// produces a nil-coalescing expression (`or 0` for int/uint64, `or ""` for string,
// `or 0.0` for float64) for each optional field's source-path argument in the
// string.format call.
//
// **Validates: Requirements 2.10, 3.4**
func TestProperty10_OptionalFieldNilCoalescing(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		pkg := genPackageName().Draw(t, "pkg")
		structName := rapid.StringMatching(`[A-Z][A-Za-z]{2,12}`).Draw(t, "structName")
		tag := rapid.StringMatching(`[a-z][a-z0-9_]{1,12}`).Draw(t, "tag")

		// Generate 2-6 non-tag fields. At least one will be optional, at least one required.
		numFields := rapid.IntRange(2, 6).Draw(t, "numFields")
		fields := make([]FieldSchema, 0, numFields+1)

		// Add a tag field.
		fields = append(fields, FieldSchema{
			Name:     "Event",
			Type:     "string",
			Position: -1,
			IsTag:    true,
		})

		fieldNames := []string{"Name", "X", "Y", "Amount", "Index", "Value", "Surface", "Player", "Unit", "Tick"}

		luaFields := make(map[string]string)
		luaDefaults := make(map[string]string)

		// Track which fields are optional and which are required.
		type fieldInfo struct {
			name       string
			goType     string
			sourcePath string
			optional   bool
		}
		var fieldInfos []fieldInfo

		// Ensure we have at least one optional and one required field.
		// First field is always required, last field is always optional.
		for i := 0; i < numFields; i++ {
			ftype := genFieldType().Draw(t, fmt.Sprintf("ftype_%d", i))
			fname := fieldNames[i]

			var isOptional bool
			if i == 0 {
				isOptional = false // Guarantee at least one required field.
			} else if i == numFields-1 {
				isOptional = true // Guarantee at least one optional field.
			} else {
				isOptional = rapid.Bool().Draw(t, fmt.Sprintf("optional_%d", i))
			}

			fields = append(fields, FieldSchema{
				Name:     fname,
				Type:     ftype,
				Position: i,
				IsTag:    false,
				Optional: isOptional,
			})

			// Generate a source path for the field.
			sourcePath := "event." + strings.ToLower(fname)
			luaFields[fname] = sourcePath

			// If optional, add a default.
			if isOptional {
				def := defaultForType(ftype)
				luaDefaults[fname] = def
			}

			fieldInfos = append(fieldInfos, fieldInfo{
				name:       fname,
				goType:     ftype,
				sourcePath: sourcePath,
				optional:   isOptional,
			})
		}

		schema := EventSchema{
			PackageName: pkg,
			PackagePath: "client/" + pkg,
			StructName:  structName,
			Tags:        []string{tag},
			Fields:      fields,
			Separator:   ":",
			Lua: &LuaSchema{
				Events:   []LuaEvent{{Name: "on_" + tag, Tag: tag}},
				Fields:   luaFields,
				Guards:   []string{},
				Tag:      tag,
				Defaults: luaDefaults,
			},
		}

		// Execute the Lua template.
		var buf bytes.Buffer
		err := luaTemplate.Execute(&buf, schema)
		if err != nil {
			t.Fatalf("template execution failed: %v", err)
		}

		output := buf.String()

		// For each optional field (has a default), verify the output contains
		// "<source_path> or <default_value>".
		for _, fi := range fieldInfos {
			if fi.optional {
				def := luaDefaults[fi.name]
				var expectedExpr string
				if def == `""` {
					expectedExpr = fi.sourcePath + ` or ""`
				} else {
					expectedExpr = fi.sourcePath + " or " + def
				}
				if !strings.Contains(output, expectedExpr) {
					t.Fatalf("optional field %s (type=%s): expected nil-coalescing expression %q not found in output.\nOutput:\n%s",
						fi.name, fi.goType, expectedExpr, output)
				}
			}
		}

		// For each required field (no default), verify that its source path
		// does NOT appear with an "or" suffix in the format arguments.
		for _, fi := range fieldInfos {
			if !fi.optional {
				// The required field's source path should appear in the output
				// without "or" suffixed to it.
				orExpr := fi.sourcePath + " or "
				if strings.Contains(output, orExpr) {
					t.Fatalf("required field %s: unexpected nil-coalescing expression %q found in output.\nOutput:\n%s",
						fi.name, orExpr, output)
				}
			}
		}
	})
}

// Feature: lua-codegen, Property 11: Iteration Field Scoping

// genLoopVarName generates a valid loop variable name (lowercase identifier).
func genLoopVarName() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{"tile", "item", "entry", "obj", "elem"})
}

// genTileFieldName generates a PascalCase Go field name for loop-scoped fields.
func genTileFieldName() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{
		"TileName", "PosX", "PosY", "TileIdx", "TileAmt",
		"TileLabel", "TileHP", "TileQual", "TileCnt", "TileSize",
	})
}

// genEventFieldName generates an event-scoped PascalCase Go field name.
func genEventFieldName() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{
		"SurfaceName", "SurfaceIndex", "PlayerIndex", "TickVal",
		"ForceName", "ReasonStr", "TargetName", "SourceId",
	})
}

// TestProperty11_IterationFieldScoping verifies that for any EventSchema with
// an iterate annotation and a declared loop variable, fields whose source path
// starts with the loop variable name are extracted inside the for loop body,
// and fields whose source path does not start with the loop variable name are
// extracted into local variables before the for loop begins.
//
// **Validates: Requirements 4.1, 4.2, 4.3**
func TestProperty11_IterationFieldScoping(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		loopVar := genLoopVarName().Draw(t, "loopVar")
		iterateSource := "event." + rapid.SampledFrom([]string{"tiles", "items", "entries", "objects"}).Draw(t, "iterSource")

		// Generate 1-3 loop-var-scoped fields (source path starts with loopVar).
		loopFieldCount := rapid.IntRange(1, 3).Draw(t, "loopFieldCount")
		// Generate 1-3 event-scoped fields (source path does NOT start with loopVar).
		eventFieldCount := rapid.IntRange(1, 3).Draw(t, "eventFieldCount")

		fields := make([]FieldSchema, 0, loopFieldCount+eventFieldCount)
		luaFields := make(map[string]string)

		loopSuffixes := []string{"name", "position.x", "position.y", "index", "amount", "label"}
		eventPaths := []string{"surface.name", "surface.index", "event.player_index", "game.tick", "event.force", "event.reason"}

		// Add loop-var-scoped fields.
		for i := 0; i < loopFieldCount; i++ {
			fieldName := genTileFieldName().Draw(t, "loopFieldName")
			sourcePath := loopVar + "." + loopSuffixes[i%len(loopSuffixes)]
			fields = append(fields, FieldSchema{
				Name:     fieldName,
				Type:     genFieldType().Draw(t, "loopFieldType"),
				Position: i,
				IsTag:    false,
				Optional: false,
			})
			luaFields[fieldName] = sourcePath
		}

		// Add event-scoped fields.
		for i := 0; i < eventFieldCount; i++ {
			fieldName := genEventFieldName().Draw(t, "eventFieldName")
			sourcePath := eventPaths[i%len(eventPaths)]
			fields = append(fields, FieldSchema{
				Name:     fieldName,
				Type:     genFieldType().Draw(t, "eventFieldType"),
				Position: loopFieldCount + i,
				IsTag:    false,
				Optional: false,
			})
			luaFields[fieldName] = sourcePath
		}

		// Build the schema with iteration pattern.
		schema := EventSchema{
			PackageName: "testpkg",
			PackagePath: "client/testpkg",
			StructName:  "TestEvent",
			Tags:        []string{"test_tag"},
			Fields:      fields,
			Lua: &LuaSchema{
				Events: []LuaEvent{
					{Name: "on_test_event_a", Tag: "test_tag"},
					{Name: "on_test_event_b", Tag: "test_tag"},
				},
				Fields:       luaFields,
				Guards:       []string{},
				Iterate:      iterateSource,
				LoopVar:      loopVar,
				Tag:          "test_tag",
				Defaults:     map[string]string{},
				IncludeCause: false,
			},
		}

		// Render the template.
		var buf strings.Builder
		err := luaTemplate.Execute(&buf, schema)
		if err != nil {
			t.Fatalf("template execution failed: %v", err)
		}
		output := buf.String()

		// Find the for loop line position.
		lines := strings.Split(output, "\n")
		forLoopLine := -1
		for i, line := range lines {
			if strings.Contains(line, "for _, "+loopVar+" in pairs("+iterateSource+")") {
				forLoopLine = i
				break
			}
		}
		if forLoopLine == -1 {
			t.Fatalf("for loop not found in output:\n%s", output)
		}

		// Property: event-scoped field assignments (local declarations) appear BEFORE the for loop.
		for fieldName, sourcePath := range luaFields {
			if strings.HasPrefix(sourcePath, loopVar+".") || strings.HasPrefix(sourcePath, loopVar+"[") {
				continue // Skip loop-scoped fields for this check.
			}
			// This is an event-scoped field; its local declaration should appear before the for loop.
			localName := snakeCase(fieldName)
			localDecl := "local " + localName + " = " + sourcePath
			declLine := -1
			for i, line := range lines {
				if strings.Contains(line, localDecl) {
					declLine = i
					break
				}
			}
			if declLine == -1 {
				t.Fatalf("event-scoped field %q (local %s) declaration not found in output:\n%s", fieldName, localName, output)
			}
			if declLine >= forLoopLine {
				t.Fatalf("event-scoped field %q (local %s) declared at line %d, but for loop is at line %d — should be before loop",
					fieldName, localName, declLine, forLoopLine)
			}
		}

		// Property: loop-var-scoped field accesses appear INSIDE the for loop (after the for line).
		for fieldName, sourcePath := range luaFields {
			if !strings.HasPrefix(sourcePath, loopVar+".") && !strings.HasPrefix(sourcePath, loopVar+"[") {
				continue // Skip event-scoped fields for this check.
			}
			// This is a loop-scoped field; its local declaration should appear after the for loop.
			localName := snakeCase(fieldName)
			localDecl := "local " + localName + " = " + sourcePath
			declLine := -1
			for i, line := range lines {
				if strings.Contains(line, localDecl) {
					declLine = i
					break
				}
			}
			if declLine == -1 {
				t.Fatalf("loop-scoped field %q (local %s) declaration not found in output:\n%s", fieldName, localName, output)
			}
			if declLine <= forLoopLine {
				t.Fatalf("loop-scoped field %q (local %s) declared at line %d, but for loop is at line %d — should be inside loop (after for line)",
					fieldName, localName, declLine, forLoopLine)
			}
		}
	})
}

// Feature: lua-codegen, Property 9: Guard Ordering Before Field Access

// TestProperty9_GuardOrderingBeforeFieldAccess verifies that for any EventSchema
// with guard annotations, the generated Lua code places all guard checks
// (of the form `if not (ref and ref.valid) then return end`) before any field
// access expression that depends on the guarded reference.
//
// **Validates: Requirements 2.4**
func TestProperty9_GuardOrderingBeforeFieldAccess(t *testing.T) {
	// Guard definitions: each guard has a source expression, a variable name,
	// and fields whose source paths reference that variable.
	type guardFieldDef struct {
		fieldName string
		path      string
		fieldType string
	}
	type guardDef struct {
		guardExpr string
		varName   string
		fields    []guardFieldDef
	}

	// Pool of guard definitions to draw from.
	guardPool := []guardDef{
		{
			guardExpr: "event.entity",
			varName:   "entity",
			fields: []guardFieldDef{
				{"EntName", "entity.name", "string"},
				{"EntUnit", "entity.unit_number", "int"},
				{"EntX", "entity.position.x", "float64"},
				{"EntY", "entity.position.y", "float64"},
			},
		},
		{
			guardExpr: "entity.surface",
			varName:   "surface",
			fields: []guardFieldDef{
				{"SurfName", "surface.name", "string"},
				{"SurfIdx", "surface.index", "int"},
			},
		},
	}

	rapid.Check(t, func(t *rapid.T) {
		// Pick 1-2 guards (using the chained entity -> surface pattern).
		numGuards := rapid.IntRange(1, 2).Draw(t, "numGuards")

		chosenGuards := guardPool[:numGuards]

		// Build the schema from chosen guards.
		guards := make([]string, numGuards)
		luaFields := make(map[string]string)
		var schemaFields []FieldSchema
		pos := 0

		// Add a tag field.
		schemaFields = append(schemaFields, FieldSchema{
			Name:     "Event",
			Type:     "string",
			Position: -1,
			IsTag:    true,
		})

		for i, g := range chosenGuards {
			guards[i] = g.guardExpr
			// Pick 1 to len(fields) fields from each guard.
			numFields := rapid.IntRange(1, len(g.fields)).Draw(t, fmt.Sprintf("numFieldsGuard%d", i))
			for j := 0; j < numFields; j++ {
				fp := g.fields[j]
				luaFields[fp.fieldName] = fp.path
				schemaFields = append(schemaFields, FieldSchema{
					Name:     fp.fieldName,
					Type:     fp.fieldType,
					Position: pos,
					IsTag:    false,
					Optional: false,
				})
				pos++
			}
		}

		schema := EventSchema{
			PackageName: "guardtest",
			PackagePath: "client/guardtest",
			StructName:  "GuardStruct",
			Tags:        []string{"guard_event"},
			Fields:      schemaFields,
			Lua: &LuaSchema{
				Events:   []LuaEvent{{Name: "on_guard_event", Tag: "guard_event"}},
				Fields:   luaFields,
				Guards:   guards,
				Tag:      "guard_event",
				Defaults: map[string]string{},
			},
		}

		// Render the template.
		var buf bytes.Buffer
		err := luaTemplate.Execute(&buf, schema)
		if err != nil {
			t.Fatalf("template execution failed: %v", err)
		}

		output := buf.String()
		lines := strings.Split(output, "\n")

		// For each guard, find the line of its validity check.
		for _, g := range chosenGuards {
			guardCheckLine := -1
			guardCheckPattern := "if not (" + g.varName + " and " + g.varName + ".valid) then return end"

			for lineNum, line := range lines {
				if strings.Contains(line, guardCheckPattern) {
					guardCheckLine = lineNum
					break
				}
			}

			if guardCheckLine == -1 {
				t.Fatalf("guard check for %q (var=%s) not found in output:\n%s",
					g.guardExpr, g.varName, output)
			}

			// For each field that references this guard variable, find its
			// access line and verify it comes after the guard check.
			for fieldName, path := range luaFields {
				// Only check fields that depend on this guard's variable.
				if !strings.HasPrefix(path, g.varName+".") && !strings.HasPrefix(path, g.varName+"[") {
					continue
				}

				// Find the first line where this field's path is accessed
				// (excluding the guard assignment and guard check lines).
				fieldAccessLine := -1
				guardAssignPattern := "local " + g.varName + " ="
				for lineNum, line := range lines {
					if strings.Contains(line, guardAssignPattern) {
						continue
					}
					if strings.Contains(line, guardCheckPattern) {
						continue
					}
					if strings.Contains(line, path) {
						fieldAccessLine = lineNum
						break
					}
				}

				if fieldAccessLine == -1 {
					// Field path not found directly — it may be inlined in format args.
					// This is acceptable as long as the format call itself is after the guard.
					continue
				}

				if guardCheckLine >= fieldAccessLine {
					t.Fatalf("PROPERTY VIOLATION: guard check for %q (var=%s) at line %d "+
						"appears AT OR AFTER field access %q (field=%s) at line %d.\nOutput:\n%s",
						g.guardExpr, g.varName, guardCheckLine,
						path, fieldName, fieldAccessLine, output)
				}
			}
		}
	})
}
