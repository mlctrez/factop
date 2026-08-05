package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

// ScanPackages walks the schema directory and returns all discovered event schemas.
// It expects subdirectories like client/player/, client/entity/, etc.
func ScanPackages(dir string) ([]EventSchema, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading schema directory %q: %w", dir, err)
	}

	var schemas []EventSchema

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pkgDir := filepath.Join(dir, entry.Name())
		pkgSchemas, err := scanPackage(pkgDir, entry.Name())
		if err != nil {
			return nil, err
		}
		schemas = append(schemas, pkgSchemas...)
	}

	return schemas, nil
}

// scanPackage scans a single package directory for event schemas.
func scanPackage(pkgDir string, pkgName string) ([]EventSchema, error) {
	fset := token.NewFileSet()

	pkgs, err := parser.ParseDir(fset, pkgDir, func(fi os.FileInfo) bool {
		// Skip test files and generated files.
		name := fi.Name()
		return !strings.HasSuffix(name, "_test.go") && !strings.HasSuffix(name, "_gen.go")
	}, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parsing package %q: %w", pkgDir, err)
	}

	var schemas []EventSchema

	for _, pkg := range pkgs {
		// Detect package-level WireSeparator constant.
		separator := detectSeparator(pkg)

		for _, file := range pkg.Files {
			fileSchemas, err := scanFile(file, pkgName, pkgDir, separator)
			if err != nil {
				return nil, err
			}
			schemas = append(schemas, fileSchemas...)
		}
	}

	return schemas, nil
}

// detectSeparator looks for a package-level constant named WireSeparator.
// Returns the default ":" if not found.
func detectSeparator(pkg *ast.Package) string {
	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.CONST {
				continue
			}
			for _, spec := range genDecl.Specs {
				valueSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, name := range valueSpec.Names {
					if name.Name == "WireSeparator" && i < len(valueSpec.Values) {
						if lit, ok := valueSpec.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
							// Unquote the string literal.
							val, err := strconv.Unquote(lit.Value)
							if err == nil {
								return val
							}
						}
					}
				}
			}
		}
	}
	return ":"
}

// scanFile scans a single Go source file for gen:event annotated structs.
func scanFile(file *ast.File, pkgName string, pkgDir string, separator string) ([]EventSchema, error) {
	var schemas []EventSchema

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// Check for gen:event annotations. Use the GenDecl's Doc first,
			// then fall back to the TypeSpec's Doc.
			var tags []string
			var luaSchema *LuaSchema
			if genDecl.Doc != nil {
				tags = extractAnnotations(genDecl.Doc)
				luaSchema = extractLuaAnnotations(genDecl.Doc)
			}
			if len(tags) == 0 && typeSpec.Doc != nil {
				tags = extractAnnotations(typeSpec.Doc)
				if luaSchema == nil {
					luaSchema = extractLuaAnnotations(typeSpec.Doc)
				}
			}

			if len(tags) == 0 {
				continue
			}

			// Extract fields with wire tags.
			fields, err := extractFields(structType)
			if err != nil {
				return nil, fmt.Errorf("struct %s in %s: %w", typeSpec.Name.Name, pkgDir, err)
			}

			// Compute relative package path from the directory structure.
			// We want something like "client/player".
			pkgPath := computePackagePath(pkgDir)

			schemas = append(schemas, EventSchema{
				PackageName: pkgName,
				PackagePath: pkgPath,
				StructName:  typeSpec.Name.Name,
				Tags:        tags,
				Fields:      fields,
				Separator:   separator,
				Lua:         luaSchema,
			})
		}
	}

	return schemas, nil
}

// extractLuaAnnotations reads // gen:lua comments from an *ast.CommentGroup.
// Returns a populated LuaSchema if any gen:lua lines are found, nil otherwise.
func extractLuaAnnotations(doc *ast.CommentGroup) *LuaSchema {
	if doc == nil {
		return nil
	}

	var schema *LuaSchema

	for _, comment := range doc.List {
		text := comment.Text
		// Strip the leading "// " prefix.
		text = strings.TrimPrefix(text, "//")
		text = strings.TrimSpace(text)

		if !strings.HasPrefix(text, "gen:lua") {
			continue
		}

		// Initialize schema on first gen:lua line.
		if schema == nil {
			schema = &LuaSchema{
				Fields:   make(map[string]string),
				Defaults: make(map[string]string),
			}
		}

		// Parse the rest after "gen:lua".
		rest := strings.TrimPrefix(text, "gen:lua")
		rest = strings.TrimSpace(rest)

		if rest == "" {
			continue
		}

		// Split into space-separated key=value pairs.
		pairs := strings.Fields(rest)
		for _, pair := range pairs {
			eqIdx := strings.Index(pair, "=")
			if eqIdx < 0 {
				// No value — store the raw key for later validation.
				schema.RawKeys = append(schema.RawKeys, pair)
				continue
			}

			key := pair[:eqIdx]
			value := pair[eqIdx+1:]

			// Track all raw keys for validation.
			schema.RawKeys = append(schema.RawKeys, key)

			switch {
			case key == "event":
				// Support event=name or event=name:tag format.
				if colonIdx := strings.Index(value, ":"); colonIdx >= 0 {
					schema.Events = append(schema.Events, LuaEvent{
						Name: value[:colonIdx],
						Tag:  value[colonIdx+1:],
					})
				} else {
					schema.Events = append(schema.Events, LuaEvent{Name: value})
				}

			case strings.HasPrefix(key, "field."):
				fieldName := strings.TrimPrefix(key, "field.")
				schema.Fields[fieldName] = value

			case key == "guard":
				schema.Guards = append(schema.Guards, value)

			case key == "iterate":
				schema.Iterate = value

			case key == "loop_var":
				schema.LoopVar = value

			case key == "tag":
				schema.Tag = value

			case strings.HasPrefix(key, "default."):
				fieldName := strings.TrimPrefix(key, "default.")
				schema.Defaults[fieldName] = value

			case key == "nth_tick":
				n, err := strconv.Atoi(value)
				if err == nil {
					schema.NthTick = n
				}

			case key == "include_cause":
				// Any non-empty value means true.
				schema.IncludeCause = value != ""

			case strings.HasPrefix(key, "fallback."):
				if schema.Fallback == nil {
					schema.Fallback = &LuaFallback{
						Defaults: make(map[string]string),
					}
				}
				subKey := strings.TrimPrefix(key, "fallback.")
				switch {
				case subKey == "object":
					schema.Fallback.Object = value
				case strings.HasPrefix(subKey, "default."):
					fieldName := strings.TrimPrefix(subKey, "default.")
					schema.Fallback.Defaults[fieldName] = value
				}
			}
		}
	}

	return schema
}

// extractAnnotations reads // gen:event comments from an *ast.CommentGroup.
// Returns the tag values found (e.g., ["move"] or ["entity_died", "entity_built"]).
func extractAnnotations(doc *ast.CommentGroup) []string {
	if doc == nil {
		return nil
	}

	var tags []string
	for _, comment := range doc.List {
		text := comment.Text
		// Strip the leading "// " prefix.
		text = strings.TrimPrefix(text, "//")
		text = strings.TrimSpace(text)

		if !strings.HasPrefix(text, "gen:event") {
			continue
		}

		// Parse "gen:event tag=<value>"
		rest := strings.TrimPrefix(text, "gen:event")
		rest = strings.TrimSpace(rest)

		if strings.HasPrefix(rest, "tag=") {
			tag := strings.TrimPrefix(rest, "tag=")
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	return tags
}

// extractFields processes struct fields and extracts wire tag information.
func extractFields(structType *ast.StructType) ([]FieldSchema, error) {
	var fields []FieldSchema

	for _, field := range structType.Fields.List {
		if field.Tag == nil {
			continue
		}

		// Parse the struct tag to find the "wire" key.
		tagValue := field.Tag.Value
		// Remove backticks.
		tagValue = strings.Trim(tagValue, "`")

		wireTag := reflect.StructTag(tagValue).Get("wire")
		if wireTag == "" {
			continue
		}

		position, isTag, optional, err := parseWireTag(wireTag)
		if err != nil {
			fieldName := ""
			if len(field.Names) > 0 {
				fieldName = field.Names[0].Name
			}
			return nil, fmt.Errorf("field %s: %w", fieldName, err)
		}

		// Skip fields with wire:"-" (no tag modifier).
		if position == -1 && !isTag {
			continue
		}

		fieldName := ""
		if len(field.Names) > 0 {
			fieldName = field.Names[0].Name
		}

		fieldType := typeString(field.Type)

		fields = append(fields, FieldSchema{
			Name:     fieldName,
			Type:     fieldType,
			Position: position,
			IsTag:    isTag,
			Optional: optional,
		})
	}

	return fields, nil
}

// parseWireTag parses a single wire struct tag value into position and modifiers.
func parseWireTag(raw string) (position int, isTag bool, optional bool, err error) {
	if raw == "" {
		return -1, false, false, nil // no wire tag, skip field
	}
	parts := strings.SplitN(raw, ",", 2)
	posStr := parts[0]

	if posStr == "-" {
		if len(parts) == 2 && parts[1] == "tag" {
			return -1, true, false, nil
		}
		return -1, false, false, nil // skip field
	}

	pos, err := strconv.Atoi(posStr)
	if err != nil || pos < 0 {
		return 0, false, false, fmt.Errorf("invalid wire position %q", posStr)
	}

	if len(parts) == 2 {
		switch parts[1] {
		case "optional":
			return pos, false, true, nil
		default:
			return 0, false, false, fmt.Errorf("unknown wire modifier %q", parts[1])
		}
	}
	return pos, false, false, nil
}

// typeString converts an ast.Expr type to its string representation.
func typeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return typeString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + typeString(t.X)
	case *ast.ArrayType:
		return "[]" + typeString(t.Elt)
	default:
		return "unknown"
	}
}

// computePackagePath extracts the relative package path from an absolute directory path.
// It looks for "client/" in the path and returns from there.
func computePackagePath(pkgDir string) string {
	// Normalize path separators.
	normalized := filepath.ToSlash(pkgDir)

	// Find "client/" in the path.
	idx := strings.LastIndex(normalized, "client/")
	if idx >= 0 {
		return normalized[idx:]
	}

	// Fallback: use last two path components.
	parts := strings.Split(normalized, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "/" + parts[len(parts)-1]
	}
	return normalized
}
