// Package main provides a code generator that reads annotated Go structs
// via go/ast and produces parser functions, format methods, OnXxx registration
// methods, and table-driven tests for wire-format event types.
//
// Usage:
//
//	go run ./gen [-schema client]
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// EventSchema represents one annotated struct extracted from source.
type EventSchema struct {
	PackageName string       // e.g., "player"
	PackagePath string       // e.g., "client/player"
	StructName  string       // e.g., "Move"
	Tags        []string     // e.g., ["move"] or ["entity_died", "entity_built", "entity_mined"]
	Fields      []FieldSchema
	Separator   string       // default ":" unless WireSeparator is found
	Lua         *LuaSchema   // nil when no gen:lua annotations present
}

// LuaSchema holds Lua-specific generation metadata parsed from // gen:lua annotations.
type LuaSchema struct {
	Events       []LuaEvent        // Factorio events to hook
	Fields       map[string]string // GoFieldName → Lua source path expression
	Guards       []string          // Variables requiring nil-and-valid checks
	Iterate      string            // Table path for iteration (e.g., "event.tiles")
	LoopVar      string            // Loop variable name (e.g., "tile")
	Tag          string            // Emitter tag override (e.g., "entity_died")
	Defaults     map[string]string // GoFieldName → default literal value
	NthTick      int               // on_nth_tick interval (0 = disabled)
	Fallback     *LuaFallback      // Fallback pattern for invalid objects
	IncludeCause bool              // Whether to include cause field logic
	RawKeys      []string          // All raw key names for validation of unrecognized keys
}

// LuaEvent represents a single Factorio event registration.
type LuaEvent struct {
	Name string // e.g., "on_entity_died"
	Tag  string // e.g., "entity_died" — the wire tag this event emits
}

// LuaFallback defines fallback behavior when an object is invalid.
type LuaFallback struct {
	Object   string            // Object to check (e.g., "surface")
	Defaults map[string]string // Field → fallback value when object is invalid
}

// FieldSchema represents one wire-tagged field.
type FieldSchema struct {
	Name     string // Go field name, e.g., "SurfaceIndex"
	Type     string // Go type: "string", "int", "float64", "uint64"
	Position int    // wire position index (0-based), or -1 for tag fields
	IsTag    bool   // true when wire:"-,tag"
	Optional bool   // true when wire:"N,optional"
}

// GeneratorConfig holds runtime configuration.
type GeneratorConfig struct {
	SchemaDir  string // default "client", overridable with -schema flag
	OutputRoot string // project root for output paths
}

func main() {
	var cfg GeneratorConfig

	flag.StringVar(&cfg.SchemaDir, "schema", "client", "directory to scan for schema structs")
	flag.Parse()

	// Determine project root from the current working directory.
	root, err := projectRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error determining project root: %v\n", err)
		os.Exit(1)
	}
	cfg.OutputRoot = root

	// Pipeline: scan → validate → render
	schemas, err := scan(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scan error: %v\n", err)
		os.Exit(1)
	}

	if err := validate(schemas); err != nil {
		fmt.Fprintf(os.Stderr, "validation error: %v\n", err)
		os.Exit(1)
	}

	if err := render(cfg, schemas); err != nil {
		fmt.Fprintf(os.Stderr, "render error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "generated code for %d schema(s)\n", len(schemas))
}

// projectRoot returns the project root directory. It walks up from the
// current working directory looking for go.mod.
func projectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found in any parent directory")
		}
		dir = parent
	}
}

// scan walks the schema directory and extracts EventSchema definitions.
func scan(cfg GeneratorConfig) ([]EventSchema, error) {
	dir := filepath.Join(cfg.OutputRoot, cfg.SchemaDir)
	return ScanPackages(dir)
}

// validate checks all schemas for correctness.
func validate(schemas []EventSchema) error {
	for _, s := range schemas {
		if err := Validate(s); err != nil {
			return err
		}
		if s.Lua != nil {
			if err := ValidateLua(s); err != nil {
				return err
			}
		}
	}
	return nil
}

// render executes templates and writes generated files.
func render(cfg GeneratorConfig, schemas []EventSchema) error {
	return RenderAll(cfg, schemas)
}
