// Package main provides a tool to fetch the Factorio runtime Lua API documentation
// and generate a filtered Markdown steering file suitable for LLM context.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	runtimeAPIURL = "https://lua-api.factorio.com/latest/runtime-api.json"
	cacheFileName = "runtime-api-cache.json"
	defaultOutput = "../.kiro/steering/factorio-api.md"
)

func main() {
	var (
		cacheDir string
		output   string
		classes  string
		events   string
		defines  string
		concepts string
		globals  bool
		list     bool
		noCache  bool
		maxAge   time.Duration
	)

	flag.StringVar(&cacheDir, "cache-dir", ".", "directory to store cached API JSON")
	flag.StringVar(&output, "output", defaultOutput, "output path for generated Markdown")
	flag.StringVar(&classes, "classes", "", "comma-separated class names to include (empty = none)")
	flag.StringVar(&events, "events", "", "comma-separated event names to include (empty = none)")
	flag.StringVar(&defines, "defines", "", "comma-separated define names to include (empty = none)")
	flag.StringVar(&concepts, "concepts", "", "comma-separated concept names to include (empty = none)")
	flag.BoolVar(&globals, "globals", false, "include global objects and functions")
	flag.BoolVar(&list, "list", false, "list all available class/event/define/concept names and exit")
	flag.BoolVar(&noCache, "no-cache", false, "bypass cache and fetch fresh data")
	flag.DurationVar(&maxAge, "max-age", 24*time.Hour, "maximum age of cached data before re-fetching")
	flag.Parse()

	api, err := loadAPI(cacheDir, noCache, maxAge)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading API: %v\n", err)
		os.Exit(1)
	}

	if list {
		printAvailable(api)
		return
	}

	filter := buildFilter(classes, events, defines, concepts, globals)
	md := generateMarkdown(api, filter)

	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating output directory: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(output, []byte(md), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %d bytes to %s\n", len(md), output)
}
