package main

import (
	"fmt"
	"sort"
	"strings"
	"text/template"
	"unicode"
)

// funcMap provides helper functions available in all templates.
var funcMap = template.FuncMap{
	"parseExpr":    parseExpr,    // returns strconv call for parsing a field
	"formatExpr":   formatExpr,   // returns strconv call for formatting a field
	"pascalCase":   pascalCase,   // "entity_died" → "EntityDied", "move" → "Move"
	"fieldCount":   fieldCount,   // count of non-tag, non-optional fields (minimum required)
	"totalFields":  totalFields,  // count of all non-tag fields (including optional)
	"exampleValue": exampleValue, // returns example literal for a field type
	"tooFewFields": tooFewFields, // returns a test string with too few fields
	"invalidField": invalidField, // returns a test string with one invalid numeric field
	"isNumeric":    isNumeric,    // returns true if type is int, float64, or uint64
	"formatArgs":   formatArgs,   // returns format arguments for Format() method
	"hasTagField":  hasTagField,  // returns true if fields include a tag field
	"tagFieldName": tagFieldName, // returns the name of the tag field
	"sub":          sub,          // integer subtraction
	"add":          add,          // integer addition
}

// parseExpr returns the Go code to parse a field from parts[N].
// For string fields, it's a direct assignment. For numeric fields,
// it includes the strconv call with error handling.
// The structName placeholder is handled by the template caller.
func parseExpr(field FieldSchema) string {
	idx := field.Position
	switch field.Type {
	case "string":
		return fmt.Sprintf("result.%s = parts[%d]", field.Name, idx)
	case "int":
		return fmt.Sprintf("result.%s, err = strconv.Atoi(parts[%d])\n\tif err != nil {\n\t\treturn result, fmt.Errorf(\"invalid %s %%q: %%w\", parts[%d], err)\n\t}", field.Name, idx, field.Name, idx)
	case "float64":
		return fmt.Sprintf("result.%s, err = strconv.ParseFloat(parts[%d], 64)\n\tif err != nil {\n\t\treturn result, fmt.Errorf(\"invalid %s %%q: %%w\", parts[%d], err)\n\t}", field.Name, idx, field.Name, idx)
	case "uint64":
		return fmt.Sprintf("result.%s, err = strconv.ParseUint(parts[%d], 10, 64)\n\tif err != nil {\n\t\treturn result, fmt.Errorf(\"invalid %s %%q: %%w\", parts[%d], err)\n\t}", field.Name, idx, field.Name, idx)
	default:
		return fmt.Sprintf("// unsupported type %s for field %s", field.Type, field.Name)
	}
}

// formatExpr returns the Go expression for formatting a field value to string.
func formatExpr(field FieldSchema) string {
	switch field.Type {
	case "string":
		return fmt.Sprintf("m.%s", field.Name)
	case "int":
		return fmt.Sprintf("strconv.Itoa(m.%s)", field.Name)
	case "float64":
		return fmt.Sprintf("strconv.FormatFloat(m.%s, 'f', 1, 64)", field.Name)
	case "uint64":
		return fmt.Sprintf("strconv.FormatUint(m.%s, 10)", field.Name)
	default:
		return fmt.Sprintf("/* unsupported type %s */", field.Type)
	}
}

// pascalCase converts underscore-separated tag names to PascalCase.
// Examples: "entity_died" → "EntityDied", "move" → "Move", "player_joined" → "PlayerJoined"
func pascalCase(s string) string {
	parts := strings.Split(s, "_")
	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			b.WriteString(part[1:])
		}
	}
	return b.String()
}

// fieldCount returns the count of non-tag, non-optional fields.
// This is the minimum number of fields required in a wire payload.
func fieldCount(fields []FieldSchema) int {
	count := 0
	for _, f := range fields {
		if !f.IsTag && !f.Optional {
			count++
		}
	}
	return count
}

// totalFields returns the count of all non-tag fields including optional.
// This is the maximum number of fields allowed in a wire payload.
func totalFields(fields []FieldSchema) int {
	count := 0
	for _, f := range fields {
		if !f.IsTag {
			count++
		}
	}
	return count
}

// exampleValue returns a Go literal example value for a field type.
// Used in generated test code.
func exampleValue(field FieldSchema) string {
	switch field.Type {
	case "string":
		return fmt.Sprintf("%q", "test"+field.Name)
	case "int":
		return "42"
	case "float64":
		return "12.5"
	case "uint64":
		return "100"
	default:
		return "nil"
	}
}

// tooFewFields returns a test input string with fewer fields than required.
// It generates a single field value (which is less than the minimum required).
func tooFewFields(fields []FieldSchema, sep string) string {
	required := fieldCount(fields)
	if required <= 1 {
		return ""
	}
	// Return only one field value.
	parts := make([]string, 1)
	parts[0] = "x"
	return strings.Join(parts, sep)
}

// invalidField returns a test input string with a specific field set to an invalid value.
// All other fields get valid placeholder values.
func invalidField(fields []FieldSchema, target FieldSchema, sep string) string {
	total := totalFields(fields)
	parts := make([]string, total)
	for _, f := range fields {
		if f.IsTag {
			continue
		}
		if f.Position < 0 || f.Position >= total {
			continue
		}
		if f.Name == target.Name {
			parts[f.Position] = "notanumber"
		} else {
			switch f.Type {
			case "string":
				parts[f.Position] = "valid"
			case "int":
				parts[f.Position] = "1"
			case "float64":
				parts[f.Position] = "1.0"
			case "uint64":
				parts[f.Position] = "1"
			default:
				parts[f.Position] = "x"
			}
		}
	}
	return strings.Join(parts, sep)
}

// isNumeric returns true if the type is int, float64, or uint64.
func isNumeric(t string) bool {
	return t == "int" || t == "float64" || t == "uint64"
}

// formatArgs returns the string concatenation expression for Format() fields.
// It joins all non-tag fields with the separator using their formatExpr.
func formatArgs(fields []FieldSchema, sep string) string {
	var exprs []string
	for _, f := range fields {
		if f.IsTag {
			continue
		}
		exprs = append(exprs, formatExpr(f))
	}
	return strings.Join(exprs, ` + "` + sep + `" + `)
}

// sub performs integer subtraction (a - b).
func sub(a, b int) int {
	return a - b
}

// add performs integer addition (a + b).
func add(a, b int) int {
	return a + b
}

// hasTagField returns true if the field list contains a tag field.
func hasTagField(fields []FieldSchema) bool {
	for _, f := range fields {
		if f.IsTag {
			return true
		}
	}
	return false
}

// tagFieldName returns the Go field name of the tag field, or empty string if none.
func tagFieldName(fields []FieldSchema) string {
	for _, f := range fields {
		if f.IsTag {
			return f.Name
		}
	}
	return ""
}

// snakeCase converts PascalCase or camelCase strings to snake_case.
// Examples: "SurfaceName" → "surface_name", "PlayerIndex" → "player_index", "X" → "x".
func snakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := rune(s[i-1])
				if unicode.IsLower(prev) {
					b.WriteRune('_')
				}
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// luaFormatSpec builds the format specifier string for a Lua string.format call.
// It iterates fields sorted by Position, skipping tag fields, and produces
// format specifiers: float64→"%.1f", int/uint64→"%d", string→"%s".
// Fields are joined with ":" separator.
func luaFormatSpec(fields []FieldSchema) string {
	// Collect non-tag fields sorted by position.
	var sorted []FieldSchema
	for _, f := range fields {
		if f.IsTag {
			continue
		}
		sorted = append(sorted, f)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Position < sorted[j].Position
	})

	specs := make([]string, 0, len(sorted))
	for _, f := range sorted {
		switch f.Type {
		case "float64":
			specs = append(specs, "%.1f")
		case "int", "uint64":
			specs = append(specs, "%d")
		case "string":
			specs = append(specs, "%s")
		default:
			specs = append(specs, "%s")
		}
	}
	return strings.Join(specs, ":")
}

// luaArgs builds the source path argument list for a Lua string.format call.
// It iterates fields sorted by wire Position (skipping tag fields), and for each
// field looks up the Lua source path from schema.Lua.Fields[field.Name].
// If the field has a default in schema.Lua.Defaults, the argument is wrapped
// with "or <default>". Returns a comma-separated argument list.
func luaArgs(schema EventSchema) string {
	if schema.Lua == nil {
		return ""
	}

	// Collect non-tag fields sorted by position.
	var sorted []FieldSchema
	for _, f := range schema.Fields {
		if f.IsTag {
			continue
		}
		sorted = append(sorted, f)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Position < sorted[j].Position
	})

	args := make([]string, 0, len(sorted))
	for _, f := range sorted {
		sourcePath := schema.Lua.Fields[f.Name]
		if sourcePath == "" {
			sourcePath = snakeCase(f.Name)
		}
		if def, ok := schema.Lua.Defaults[f.Name]; ok {
			if def == "" {
				sourcePath = sourcePath + ` or ""`
			} else {
				sourcePath = sourcePath + " or " + def
			}
		}
		args = append(args, sourcePath)
	}
	return strings.Join(args, ", ")
}

// luaFuncMap provides helper functions for Lua template rendering.
var luaFuncMap = template.FuncMap{
	"luaFormatSpec":           luaFormatSpec,
	"luaArgs":                 luaArgs,
	"snakeCase":               snakeCase,
	"luaSharedEmit":           luaSharedEmit,
	"luaEventTag":             luaEventTag,
	"luaHasIncludeCauseEvent": luaHasIncludeCauseEvent,
	"luaGuardVar":             luaGuardVar,
	"luaLocalFields":          luaLocalFields,
	"luaPreLoopFields":        luaPreLoopFields,
	"luaLoopFields":           luaLoopFields,
	"luaIterArgs":             luaIterArgs,
	"last":                    last,
	"luaFallbackTag":          luaFallbackTag,
	"luaFallbackLookup":       luaFallbackLookup,
	"luaValidBranchArgs":      luaValidBranchArgs,
	"luaFallbackArgs":         luaFallbackArgs,
	"luaFallbackFormatSpec":   luaFallbackFormatSpec,
	"luaIsFallbackEvent":      luaIsFallbackEvent,
	"luaSimpleHandlerBody":    luaSimpleHandlerBody,
	"luaSharedEmitLocals":     luaSharedEmitLocals,
	"luaPreLoopLocals":        luaPreLoopLocals,
	"luaLoopBody":             luaLoopBody,
	"luaFormatArgs":           luaFormatArgs,
}

// luaSharedEmit returns true if the schema should use a shared emit function.
// This happens when multiple events map to shared tags (e.g., entity, tile).
func luaSharedEmit(schema EventSchema) bool {
	if schema.Lua == nil || len(schema.Lua.Events) <= 1 {
		return false
	}
	// Use shared emit if any event has a Tag field set (explicit tag mapping)
	// OR if IncludeCause is true (entity pattern) OR if Iterate is set (tile pattern).
	if schema.Lua.IncludeCause || schema.Lua.Iterate != "" {
		return true
	}
	// Check if multiple events share the same tag.
	tags := make(map[string]int)
	for _, ev := range schema.Lua.Events {
		tag := luaEventTag(schema, ev)
		tags[tag]++
	}
	for _, count := range tags {
		if count > 1 {
			return true
		}
	}
	return false
}

// luaEventTag returns the wire tag for a given LuaEvent within a schema.
// It checks the event's Tag field first, then falls back to matching against
// EventSchema.Tags, and finally uses the schema's Lua.Tag or first tag.
func luaEventTag(schema EventSchema, event LuaEvent) string {
	// If the event has an explicit tag, use it.
	if event.Tag != "" {
		return event.Tag
	}
	// Fall back to EventSchema.Tags if there's only one.
	if len(schema.Tags) == 1 {
		return schema.Tags[0]
	}
	// Try to match event name to a tag by checking if the tag appears in the event name.
	for _, tag := range schema.Tags {
		// Check if event name contains tag with underscores replaced.
		if strings.Contains(event.Name, tag) {
			return tag
		}
	}
	// Fall back to LuaSchema.Tag if set.
	if schema.Lua != nil && schema.Lua.Tag != "" {
		return schema.Lua.Tag
	}
	// Last resort: first tag.
	if len(schema.Tags) > 0 {
		return schema.Tags[0]
	}
	return ""
}

// luaHasIncludeCauseEvent returns true if the given event should include cause.
// For entity events, only the first tag (entity_died) includes cause.
func luaHasIncludeCauseEvent(schema EventSchema, event LuaEvent) bool {
	if schema.Lua == nil || !schema.Lua.IncludeCause {
		return false
	}
	tag := luaEventTag(schema, event)
	// include_cause applies to the first tag in the schema.
	if len(schema.Tags) > 0 {
		return tag == schema.Tags[0]
	}
	return false
}

// luaGuardField is a local variable extracted from a guard expression.
type luaGuardField struct {
	LocalName string
	Expr      string
}

// luaGuardVar extracts the local variable name from a guard expression.
// E.g., "event.entity" → "entity", "event.entity.surface" → "surface",
// "game.surfaces[event.surface_index]" → "surface"
func luaGuardVar(guard string) string {
	// Handle bracket syntax like "game.surfaces[event.surface_index]"
	if idx := strings.Index(guard, "["); idx >= 0 {
		// Get the part before the bracket
		prefix := guard[:idx]
		parts := strings.Split(prefix, ".")
		lastPart := parts[len(parts)-1]
		// Singularize: "surfaces" → "surface", "players" → "player"
		if strings.HasSuffix(lastPart, "s") {
			return lastPart[:len(lastPart)-1]
		}
		return lastPart
	}
	// Handle dot syntax like "event.entity.surface"
	parts := strings.Split(guard, ".")
	return parts[len(parts)-1]
}

// luaLocalFields returns a list of fields that need local variable extraction
// in the shared emit function (for entity-style patterns with defaults).
func luaLocalFields(schema EventSchema) []luaGuardField {
	if schema.Lua == nil {
		return nil
	}

	var fields []luaGuardField
	for fieldName, def := range schema.Lua.Defaults {
		if fieldName == "Cause" && schema.Lua.IncludeCause {
			// Cause is handled separately in include_cause block.
			continue
		}
		sourcePath := schema.Lua.Fields[fieldName]
		if sourcePath == "" {
			continue
		}
		localName := snakeCase(fieldName)
		fields = append(fields, luaGuardField{
			LocalName: localName,
			Expr:      sourcePath + " or " + def,
		})
	}

	// Sort for deterministic output.
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].LocalName < fields[j].LocalName
	})
	return fields
}

// luaPreLoopFields returns fields that should be extracted before the iteration loop.
// These are event-scoped fields (their source path does NOT start with the loop variable).
func luaPreLoopFields(schema EventSchema) []luaGuardField {
	if schema.Lua == nil || schema.Lua.Iterate == "" {
		return nil
	}

	loopVar := schema.Lua.LoopVar
	var fields []luaGuardField

	// Collect fields whose source path doesn't start with the loop variable.
	for fieldName, sourcePath := range schema.Lua.Fields {
		if strings.HasPrefix(sourcePath, loopVar+".") || strings.HasPrefix(sourcePath, loopVar+"[") {
			continue
		}

		localName := snakeCase(fieldName)
		expr := sourcePath
		if def, ok := schema.Lua.Defaults[fieldName]; ok {
			expr = expr + " or " + def
		}
		fields = append(fields, luaGuardField{
			LocalName: localName,
			Expr:      expr,
		})
	}

	// Sort for deterministic output.
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].LocalName < fields[j].LocalName
	})
	return fields
}

// luaLoopFields returns fields that should be extracted inside the iteration loop.
// These are loop-variable-scoped fields (their source path starts with the loop variable).
func luaLoopFields(schema EventSchema) []luaGuardField {
	if schema.Lua == nil || schema.Lua.Iterate == "" {
		return nil
	}

	loopVar := schema.Lua.LoopVar
	var fields []luaGuardField

	for fieldName, sourcePath := range schema.Lua.Fields {
		if !strings.HasPrefix(sourcePath, loopVar+".") && !strings.HasPrefix(sourcePath, loopVar+"[") {
			continue
		}
		localName := snakeCase(fieldName)
		fields = append(fields, luaGuardField{
			LocalName: localName,
			Expr:      sourcePath,
		})
	}

	// Sort for deterministic output.
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].LocalName < fields[j].LocalName
	})
	return fields
}

// luaIterArgs returns the argument list for string.format inside an iteration loop.
// Fields that were extracted to local variables (pre-loop or in-loop) use their local names.
func luaIterArgs(schema EventSchema) []string {
	if schema.Lua == nil {
		return nil
	}

	loopVar := schema.Lua.LoopVar

	// Collect non-tag fields sorted by position.
	var sorted []FieldSchema
	for _, f := range schema.Fields {
		if f.IsTag {
			continue
		}
		sorted = append(sorted, f)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Position < sorted[j].Position
	})

	args := make([]string, 0, len(sorted))
	for _, f := range sorted {
		sourcePath := schema.Lua.Fields[f.Name]
		if sourcePath == "" {
			sourcePath = snakeCase(f.Name)
		}
		localName := snakeCase(f.Name)

		// All fields are extracted to locals (pre-loop or in-loop).
		// The default is already applied in the local declaration, so just use the name.
		if strings.HasPrefix(sourcePath, loopVar+".") || strings.HasPrefix(sourcePath, loopVar+"[") {
			// Loop-scoped
			args = append(args, localName)
		} else {
			// Pre-loop-scoped (default already in local declaration)
			args = append(args, localName)
		}
	}
	return args
}

// last returns true if index i is the last element in the slice.
func last(i int, slice []string) bool {
	return i == len(slice)-1
}

// luaFallbackTag returns the tag that has the fallback pattern applied.
// For surface, this is "surface_deleted" (the second tag).
func luaFallbackTag(schema EventSchema) string {
	if schema.Lua == nil || schema.Lua.Fallback == nil {
		return ""
	}
	// The fallback tag is typically the last tag (e.g., surface_deleted).
	if len(schema.Tags) > 1 {
		return schema.Tags[len(schema.Tags)-1]
	}
	if len(schema.Tags) == 1 {
		return schema.Tags[0]
	}
	return ""
}

// luaFallbackLookup returns the Lua expression to look up the fallback object.
// For surface, this is "game.surfaces[event.surface_index]".
func luaFallbackLookup(schema EventSchema) string {
	if schema.Lua == nil || schema.Lua.Fallback == nil {
		return ""
	}
	// Find the guard that references the fallback object.
	for _, g := range schema.Lua.Guards {
		if strings.Contains(g, schema.Lua.Fallback.Object) || luaGuardVar(g) == schema.Lua.Fallback.Object {
			return g
		}
	}
	return schema.Lua.Fallback.Object
}

// luaValidBranchArgs returns the argument list for the valid branch of a fallback.
func luaValidBranchArgs(schema EventSchema) string {
	return luaArgs(schema)
}

// luaFallbackArgs returns the argument list for the fallback branch.
func luaFallbackArgs(schema EventSchema) string {
	if schema.Lua == nil || schema.Lua.Fallback == nil {
		return ""
	}

	// Collect non-tag fields sorted by position.
	var sorted []FieldSchema
	for _, f := range schema.Fields {
		if f.IsTag {
			continue
		}
		sorted = append(sorted, f)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Position < sorted[j].Position
	})

	args := make([]string, 0, len(sorted))
	for _, f := range sorted {
		if fbVal, ok := schema.Lua.Fallback.Defaults[f.Name]; ok {
			args = append(args, fbVal)
		} else {
			sourcePath := schema.Lua.Fields[f.Name]
			if sourcePath == "" {
				sourcePath = snakeCase(f.Name)
			}
			if def, ok := schema.Lua.Defaults[f.Name]; ok {
				if def == "" {
					sourcePath = sourcePath + ` or ""`
				} else {
					sourcePath = sourcePath + " or " + def
				}
			}
			args = append(args, sourcePath)
		}
	}
	return strings.Join(args, ", ")
}

// luaFallbackFormatSpec returns the format specifier for the fallback branch.
// It may differ from the normal format spec if fallback defaults change types.
func luaFallbackFormatSpec(schema EventSchema) string {
	if schema.Lua == nil || schema.Lua.Fallback == nil {
		return luaFormatSpec(schema.Fields)
	}

	// Collect non-tag fields sorted by position.
	var sorted []FieldSchema
	for _, f := range schema.Fields {
		if f.IsTag {
			continue
		}
		sorted = append(sorted, f)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Position < sorted[j].Position
	})

	specs := make([]string, 0, len(sorted))
	for _, f := range sorted {
		// Check if this field has a fallback default that changes the format.
		if fbVal, ok := schema.Lua.Fallback.Defaults[f.Name]; ok {
			// If fallback is empty string literal, use %s
			if fbVal == `""` || fbVal == "" {
				specs = append(specs, "%s")
				continue
			}
			// If fallback is an expression that could be int, keep original type
		}
		switch f.Type {
		case "float64":
			specs = append(specs, "%.1f")
		case "int", "uint64":
			specs = append(specs, "%d")
		case "string":
			specs = append(specs, "%s")
		default:
			specs = append(specs, "%s")
		}
	}
	return strings.Join(specs, ":")
}

// luaIsFallbackEvent returns true if this event should use the fallback pattern.
func luaIsFallbackEvent(schema EventSchema, event LuaEvent) bool {
	if schema.Lua == nil || schema.Lua.Fallback == nil {
		return false
	}
	tag := luaEventTag(schema, event)
	return tag == luaFallbackTag(schema)
}

// luaSimpleHandlerBody generates the body of a simple handler function
// (guards, format, send) as a string.
func luaSimpleHandlerBody(schema EventSchema, event LuaEvent) string {
	if schema.Lua == nil {
		return ""
	}

	var b strings.Builder
	tag := luaEventTag(schema, event)

	// Guard checks
	for _, g := range schema.Lua.Guards {
		gVar := luaGuardVar(g)
		b.WriteString(fmt.Sprintf("    local %s = %s\n", gVar, g))
		b.WriteString(fmt.Sprintf("    if not (%s and %s.valid) then return end\n", gVar, gVar))
		b.WriteString("\n")
	}

	// Build args
	args := luaArgs(schema)

	// Format and send - use multi-line for readability if many args
	argList := strings.Split(args, ", ")
	if len(argList) <= 3 {
		b.WriteString(fmt.Sprintf("    local msg = string.format(\"[%s] %s\", %s)\n",
			tag, luaFormatSpec(schema.Fields), args))
	} else {
		b.WriteString(fmt.Sprintf("    local msg = string.format(\"[%s] %s\",\n",
			tag, luaFormatSpec(schema.Fields)))
		for i, arg := range argList {
			if i == len(argList)-1 {
				b.WriteString(fmt.Sprintf("        %s)\n", arg))
			} else {
				b.WriteString(fmt.Sprintf("        %s,\n", arg))
			}
		}
	}
	b.WriteString("    helpers.send_udp(4000, msg, 0)")

	return b.String()
}

// luaSharedEmitLocals generates local variable declarations for the shared emit body.
// These are fields with defaults that need local variable extraction.
func luaSharedEmitLocals(schema EventSchema) string {
	if schema.Lua == nil {
		return ""
	}

	fields := luaLocalFields(schema)
	if len(fields) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n")
	for _, f := range fields {
		b.WriteString(fmt.Sprintf("    local %s = %s\n", f.LocalName, f.Expr))
	}
	return b.String()
}

// luaPreLoopLocals generates local variable declarations before the iteration loop.
func luaPreLoopLocals(schema EventSchema) string {
	if schema.Lua == nil || schema.Lua.Iterate == "" {
		return ""
	}

	fields := luaPreLoopFields(schema)
	if len(fields) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n")
	for _, f := range fields {
		b.WriteString(fmt.Sprintf("    local %s = %s\n", f.LocalName, f.Expr))
	}
	b.WriteString("\n")
	return b.String()
}

// luaLoopBody generates the body of the iteration for loop.
func luaLoopBody(schema EventSchema) string {
	if schema.Lua == nil || schema.Lua.Iterate == "" {
		return ""
	}

	var b strings.Builder

	// Loop-scoped local variables
	loopFields := luaLoopFields(schema)
	for _, f := range loopFields {
		b.WriteString(fmt.Sprintf("        local %s = %s\n", f.LocalName, f.Expr))
	}

	// Format call
	iterArgs := luaIterArgs(schema)
	b.WriteString(fmt.Sprintf("        local msg = string.format(\"[%%s] %s\",\n", luaFormatSpec(schema.Fields)))
	b.WriteString("            tag,\n")
	for i, arg := range iterArgs {
		if i == len(iterArgs)-1 {
			b.WriteString(fmt.Sprintf("            %s)\n", arg))
		} else {
			b.WriteString(fmt.Sprintf("            %s,\n", arg))
		}
	}
	b.WriteString("        helpers.send_udp(4000, msg, 0)\n")

	return b.String()
}

// luaFormatArgs returns the comma-separated argument list for the shared emit
// string.format call (without the tag argument which is handled by the template).
// For fields with defaults that have been extracted to locals, it uses the local name.
// For fields without defaults, it uses the source path directly.
func luaFormatArgs(schema EventSchema) string {
	if schema.Lua == nil {
		return ""
	}

	// Collect non-tag fields sorted by position.
	var sorted []FieldSchema
	for _, f := range schema.Fields {
		if f.IsTag {
			continue
		}
		sorted = append(sorted, f)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Position < sorted[j].Position
	})

	args := make([]string, 0, len(sorted))
	for _, f := range sorted {
		// If field has include_cause handling, use "cause" local
		if f.Name == "Cause" && schema.Lua.IncludeCause {
			args = append(args, "cause")
			continue
		}
		// If field has a default, use the local variable name
		if _, hasDefault := schema.Lua.Defaults[f.Name]; hasDefault {
			args = append(args, snakeCase(f.Name))
			continue
		}
		// Otherwise use the source path
		sourcePath := schema.Lua.Fields[f.Name]
		if sourcePath == "" {
			sourcePath = snakeCase(f.Name)
		}
		args = append(args, sourcePath)
	}
	return strings.Join(args, ",\n        ")
}

// parserTemplate generates Parse<StructName> functions with strconv calls.
var parserTemplate = template.Must(template.New("parser").Funcs(funcMap).Parse(parserTmpl))

const parserTmpl = `// Code generated by go run ./gen -- DO NOT EDIT.

package {{.PackageName}}

import (
	"fmt"
	"strconv"
	"strings"
)

{{range $tag := .Tags}}
// Parse{{$.StructName}}{{if gt (len $.Tags) 1}}{{pascalCase $tag}}{{end}} parses a [{{$tag}}] event message into a {{$.StructName}}.
func Parse{{$.StructName}}{{if gt (len $.Tags) 1}}{{pascalCase $tag}}{{end}}(msg string) ({{$.StructName}}, error) {
	payload := msg
	prefix := "[{{$tag}}] "
	if strings.HasPrefix(msg, prefix) {
		payload = msg[len(prefix):]
	}

	parts := strings.Split(payload, "{{$.Separator}}")
	if len(parts) < {{fieldCount $.Fields}} {
		return {{$.StructName}}{}, fmt.Errorf("expected {{fieldCount $.Fields}} {{$.Separator}}-separated fields, got %d", len(parts))
	}
	if len(parts) > {{totalFields $.Fields}} {
		return {{$.StructName}}{}, fmt.Errorf("expected {{totalFields $.Fields}} {{$.Separator}}-separated fields, got %d", len(parts))
	}

	var result {{$.StructName}}
	var err error
	_ = err
{{range $.Fields}}{{if .IsTag}}
	result.{{.Name}} = "{{$tag}}"
{{else if not .Optional}}
	{{parseExpr .}}
{{end}}{{end}}
{{range $.Fields}}{{if .Optional}}	if len(parts) > {{.Position}} {
		{{parseExpr .}}
	}
{{end}}{{end}}
	return result, nil
}
{{end}}
`

// formatTemplate generates the Format() method on the struct.
var formatTemplate = template.Must(template.New("format").Funcs(funcMap).Parse(formatTmpl))

const formatTmpl = `
// Format returns the wire-format string for this {{.StructName}}.
func (m {{.StructName}}) Format() string {
{{- if hasTagField .Fields}}
	tag := m.{{tagFieldName .Fields}}
{{- else}}
	tag := "{{index .Tags 0}}"
{{- end}}
	return "[" + tag + "] " + {{formatArgs .Fields .Separator}}
}
`

// registrationTemplate generates On<PascalCaseTag> methods in the plugin package.
var registrationTemplate = template.Must(template.New("registration").Funcs(funcMap).Parse(registrationTmpl))

const registrationTmpl = `// Code generated by go run ./gen -- DO NOT EDIT.

package plugin

import "github.com/mlctrez/factop/{{.PackagePath}}"

{{range $tag := .Tags}}
// On{{pascalCase $tag}} registers a handler for [{{$tag}}] events.
func (r *EventRouter) On{{pascalCase $tag}}(fn func({{$.PackageName}}.{{$.StructName}})) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.subscribe("{{$tag}}"); err != nil {
		return err
	}

	ts := r.tags["{{$tag}}"]
	found := false
	for i := range ts.typed {
		if ts.typed[i].parse != nil {
			ts.typed[i].handlers = append(ts.typed[i].handlers, func(v any) {
				fn(v.({{$.PackageName}}.{{$.StructName}}))
			})
			found = true
			break
		}
	}
	if !found {
		ts.typed = append(ts.typed, typedHandler{
			parse: func(data []byte) (any, error) {
				return {{$.PackageName}}.Parse{{$.StructName}}{{if gt (len $.Tags) 1}}{{pascalCase $tag}}{{end}}(string(data))
			},
			handlers: []func(any){func(v any) { fn(v.({{$.PackageName}}.{{$.StructName}})) }},
		})
	}
	return nil
}
{{end}}
`

// testTemplate generates round-trip and error-case tests.
var testTemplate = template.Must(template.New("test").Funcs(funcMap).Parse(testTmpl))

const testTmpl = `// Code generated by go run ./gen -- DO NOT EDIT.

package {{.PackageName}}

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

{{range $tag := .Tags}}
func TestParse{{$.StructName}}{{if gt (len $.Tags) 1}}{{pascalCase $tag}}{{end}}_RoundTrip(t *testing.T) {
	input := {{$.StructName}}{
{{range $.Fields}}{{if .IsTag}}		{{.Name}}: "{{$tag}}",
{{else}}		{{.Name}}: {{exampleValue .}},
{{end}}{{end}}	}
	formatted := input.Format()
	parsed, err := Parse{{$.StructName}}{{if gt (len $.Tags) 1}}{{pascalCase $tag}}{{end}}(formatted)
	require.NoError(t, err)
{{range $.Fields}}{{if eq .Type "float64"}}	assert.InDelta(t, input.{{.Name}}, parsed.{{.Name}}, 0.001)
{{else if not .IsTag}}	assert.Equal(t, input.{{.Name}}, parsed.{{.Name}})
{{end}}{{end}}}

func TestParse{{$.StructName}}{{if gt (len $.Tags) 1}}{{pascalCase $tag}}{{end}}_Errors(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{name: "empty string", input: ""},
		{name: "too few fields", input: "{{tooFewFields $.Fields $.Separator}}"},
{{range $.Fields}}{{if isNumeric .Type}}		{name: "invalid {{.Name}}", input: "{{invalidField $.Fields . $.Separator}}"},
{{end}}{{end}}	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse{{$.StructName}}{{if gt (len $.Tags) 1}}{{pascalCase $tag}}{{end}}(tc.input)
			assert.Error(t, err)
		})
	}
}
{{end}}
`

// luaTemplate generates the Lua event emitter module.
var luaTemplate = template.Must(template.New("lua").Funcs(funcMap).Funcs(luaFuncMap).Parse(luaTmpl))

const luaTmpl = `-- Code generated by go run ./gen -- DO NOT EDIT.

local {{.PackageName}}_mod = {}
{{- if and .Lua (gt .Lua.NthTick 0)}}

local TICK_INTERVAL = {{.Lua.NthTick}}

local function on_tick(event)
    local msg = string.format("[{{index .Tags 0}}] {{luaFormatSpec .Fields}}", {{luaArgs .}})
    helpers.send_udp(4000, msg, 0)
end

{{.PackageName}}_mod.on_nth_tick = {
    [TICK_INTERVAL] = on_tick,
}
{{- else if and .Lua (luaSharedEmit .)}}
{{- if .Lua.Iterate}}

local function handle_{{.PackageName}}_event(event, tag)
{{- range .Lua.Guards}}
    local {{luaGuardVar .}} = {{.}}
    if not ({{luaGuardVar .}} and {{luaGuardVar .}}.valid) then return end
{{ end}}
{{- luaPreLoopLocals .}}    for _, {{.Lua.LoopVar}} in pairs({{.Lua.Iterate}}) do
{{luaLoopBody .}}    end
end
{{- else}}

local function emit_{{.PackageName}}(tag, event{{- if .Lua.IncludeCause}}, include_cause{{end}})
{{- range .Lua.Guards}}
    local {{luaGuardVar .}} = {{.}}
    if not ({{luaGuardVar .}} and {{luaGuardVar .}}.valid) then return end
{{ end}}
{{- luaSharedEmitLocals .}}{{- if .Lua.IncludeCause}}
    local cause = ""
    if include_cause and event.cause and event.cause.valid then
        cause = event.cause.name
    end
{{end}}
    local msg = string.format("[%s] {{luaFormatSpec .Fields}}",
        tag,
        {{luaFormatArgs .}})

    helpers.send_udp(4000, msg, 0)
end
{{- end}}
{{range .Lua.Events}}
local function {{.Name}}(event)
{{- if $.Lua.Iterate}}
    handle_{{$.PackageName}}_event(event, "{{luaEventTag $ .}}")
{{- else}}
    emit_{{$.PackageName}}("{{luaEventTag $ .}}", event{{- if $.Lua.IncludeCause}}, {{luaHasIncludeCauseEvent $ .}}{{end}})
{{- end}}
end
{{end}}
{{.PackageName}}_mod.events = {
{{- range .Lua.Events}}
    [defines.events.{{.Name}}] = {{.Name}},
{{- end}}
}
{{- else if .Lua}}
{{range .Lua.Events}}
local function {{.Name}}(event)
{{- if and $.Lua.Fallback (luaIsFallbackEvent $ .)}}
    local {{$.Lua.Fallback.Object}} = {{luaFallbackLookup $}}
    if {{$.Lua.Fallback.Object}} and {{$.Lua.Fallback.Object}}.valid then
        local msg = string.format("[{{luaEventTag $ .}}] {{luaFormatSpec $.Fields}}", {{luaValidBranchArgs $}})
        helpers.send_udp(4000, msg, 0)
    else
        local msg = string.format("[{{luaEventTag $ .}}] {{luaFallbackFormatSpec $}}", {{luaFallbackArgs $}})
        helpers.send_udp(4000, msg, 0)
    end
{{- else}}
{{luaSimpleHandlerBody $ .}}{{- end}}
end
{{end}}
{{.PackageName}}_mod.events = {
{{- range .Lua.Events}}
    [defines.events.{{.Name}}] = {{.Name}},
{{- end}}
}
{{- end}}

return {{.PackageName}}_mod
`
