package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/tools/imports"
)

// RenderAll orchestrates rendering for all schemas, producing three output
// files per schema: parser+format, registration methods, and tests.
func RenderAll(cfg GeneratorConfig, schemas []EventSchema) error {
	for _, schema := range schemas {
		if err := renderSchema(cfg, schema); err != nil {
			return err
		}
	}
	return nil
}

// renderSchema renders the three generated files for a single EventSchema.
func renderSchema(cfg GeneratorConfig, schema EventSchema) error {
	// 1. Render parser + format file: client/<module>/<module>_gen.go
	parserPath := filepath.Join(cfg.OutputRoot, schema.PackagePath, schema.PackageName+"_gen.go")
	if err := renderParserFormat(parserPath, schema); err != nil {
		return fmt.Errorf("rendering parser+format for %s: %w", schema.StructName, err)
	}

	// 2. Render registration file: plugin/events_<module>_gen.go
	registrationPath := filepath.Join(cfg.OutputRoot, "plugin", "events_"+schema.PackageName+"_gen.go")
	if err := renderRegistration(registrationPath, schema); err != nil {
		return fmt.Errorf("rendering registration for %s: %w", schema.StructName, err)
	}

	// 3. Render test file: client/<module>/<module>_gen_test.go
	testPath := filepath.Join(cfg.OutputRoot, schema.PackagePath, schema.PackageName+"_gen_test.go")
	if err := renderTest(testPath, schema); err != nil {
		return fmt.Errorf("rendering tests for %s: %w", schema.StructName, err)
	}

	// 4. Render Lua emitter (only when gen:lua annotations are present)
	if schema.Lua != nil {
		luaPath := filepath.Join(cfg.OutputRoot, "softmod", "factop", schema.PackageName+"_gen.lua")
		if err := renderLua(luaPath, schema); err != nil {
			return fmt.Errorf("rendering lua for %s: %w", schema.StructName, err)
		}
	}

	return nil
}

// renderParserFormat renders the parser and format templates together into a single file.
func renderParserFormat(path string, schema EventSchema) error {
	var buf bytes.Buffer

	// Execute parser template (includes package declaration and imports)
	if err := parserTemplate.Execute(&buf, schema); err != nil {
		return fmt.Errorf("executing parser template: %w", err)
	}

	// Execute format template (appended to same file)
	if err := formatTemplate.Execute(&buf, schema); err != nil {
		return fmt.Errorf("executing format template: %w", err)
	}

	return formatAndWrite(path, buf.Bytes())
}

// renderRegistration renders the registration template into a file.
func renderRegistration(path string, schema EventSchema) error {
	var buf bytes.Buffer

	if err := registrationTemplate.Execute(&buf, schema); err != nil {
		return fmt.Errorf("executing registration template: %w", err)
	}

	return formatAndWrite(path, buf.Bytes())
}

// renderTest renders the test template into a file.
func renderTest(path string, schema EventSchema) error {
	var buf bytes.Buffer

	if err := testTemplate.Execute(&buf, schema); err != nil {
		return fmt.Errorf("executing test template: %w", err)
	}

	return formatAndWrite(path, buf.Bytes())
}

// renderLua executes the Lua template and writes the output to path.
// It skips writing if the file already exists with identical content (idempotent).
func renderLua(path string, schema EventSchema) error {
	var buf bytes.Buffer

	if err := luaTemplate.Execute(&buf, schema); err != nil {
		return fmt.Errorf("executing lua template for %s: %w", path, err)
	}

	newContent := buf.Bytes()

	// Ensure output directory exists.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	// Idempotent write: skip if existing content matches.
	existing, err := os.ReadFile(path)
	if err == nil && bytes.Equal(existing, newContent) {
		return nil
	}

	if err := os.WriteFile(path, newContent, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}

// formatAndWrite processes raw template output through goimports (which also
// handles gofmt formatting) and writes the result to disk with 0644 permissions.
func formatAndWrite(path string, src []byte) error {
	// Use golang.org/x/tools/imports.Process which handles both formatting
	// and import management (adds missing imports, removes unused ones).
	formatted, err := imports.Process(path, src, nil)
	if err != nil {
		return fmt.Errorf("formatting %s: %w", path, err)
	}

	// Ensure the output directory exists.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	// Write the formatted source to disk.
	if err := os.WriteFile(path, formatted, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}
