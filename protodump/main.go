// Package main provides a tool that parses Factorio's data-raw-dump.json
// and generates a Go source file containing all prototype names grouped by type.
//
// Usage:
//
//	# First, generate the dump (one-time or when Factorio updates):
//	/opt/factorio/2.0.76/bin/x64/factorio --dump-data
//
//	# Then generate the Go file:
//	go run ./protodump --factorio-dir /opt/factorio/2.0.76 --output client/prototypes/prototypes_gen.go
//
// The generated file contains typed string constants and lookup maps for
// validating entity names, tile names, and other prototype IDs before
// sending them to the server.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const dumpRelPath = "script-output/data-raw-dump.json"

func main() {
	var (
		factorioDir string
		dumpPath    string
		output      string
		types       string
	)

	flag.StringVar(&factorioDir, "factorio-dir", "", "path to Factorio install (e.g. /opt/factorio/2.0.76)")
	flag.StringVar(&dumpPath, "dump-path", "", "direct path to data-raw-dump.json (overrides factorio-dir)")
	flag.StringVar(&output, "output", "client/prototype/prototype_gen.go", "output Go file path")
	flag.StringVar(&types, "types", "", "comma-separated prototype types to include (empty = all entity-like + tile)")
	flag.Parse()

	if dumpPath == "" {
		if factorioDir == "" {
			fmt.Fprintln(os.Stderr, "either --factorio-dir or --dump-path is required")
			os.Exit(1)
		}
		dumpPath = filepath.Join(factorioDir, dumpRelPath)
	}

	data, err := os.ReadFile(dumpPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading dump: %v\n", err)
		os.Exit(1)
	}

	var raw map[string]map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		fmt.Fprintf(os.Stderr, "parsing dump: %v\n", err)
		os.Exit(1)
	}

	filter := buildTypeFilter(types)
	groups := extractGroups(raw, filter)

	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "creating output dir: %v\n", err)
		os.Exit(1)
	}

	if err := writeGoFile(output, groups); err != nil {
		fmt.Fprintf(os.Stderr, "writing output: %v\n", err)
		os.Exit(1)
	}

	total := 0
	for _, g := range groups {
		total += len(g.Names)
	}
	fmt.Fprintf(os.Stderr, "wrote %d prototypes across %d types to %s\n", total, len(groups), output)
}
