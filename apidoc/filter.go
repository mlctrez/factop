package main

import (
	"fmt"
	"os"
	"slices"
	"strings"
)

// Filter controls which parts of the API are included in the output.
type Filter struct {
	Classes  []string
	Events   []string
	Defines  []string
	Concepts []string
	Globals  bool
}

func buildFilter(classes, events, defines, concepts string, globals bool) Filter {
	return Filter{
		Classes:  splitCSV(classes),
		Events:   splitCSV(events),
		Defines:  splitCSV(defines),
		Concepts: splitCSV(concepts),
		Globals:  globals,
	}
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (f Filter) isEmpty() bool {
	return len(f.Classes) == 0 && len(f.Events) == 0 &&
		len(f.Defines) == 0 && len(f.Concepts) == 0 && !f.Globals
}

func printAvailable(api *RuntimeAPI) {
	fmt.Fprintln(os.Stderr, "Classes:")
	for _, c := range api.Classes {
		parent := ""
		if c.Parent != "" {
			parent = " (extends " + c.Parent + ")"
		}
		fmt.Fprintf(os.Stderr, "  %s%s - %d methods, %d attributes\n",
			c.Name, parent, len(c.Methods), len(c.Attributes))
	}

	fmt.Fprintln(os.Stderr, "\nEvents:")
	for _, e := range api.Events {
		fmt.Fprintf(os.Stderr, "  %s\n", e.Name)
	}

	fmt.Fprintln(os.Stderr, "\nDefines:")
	printDefines(api.Defines, "  ")

	fmt.Fprintln(os.Stderr, "\nConcepts:")
	for _, c := range api.Concepts {
		fmt.Fprintf(os.Stderr, "  %s\n", c.Name)
	}

	fmt.Fprintln(os.Stderr, "\nGlobal Objects:")
	for _, g := range api.GlobalObjects {
		fmt.Fprintf(os.Stderr, "  %s :: %s\n", g.Name, formatType(g.Type))
	}

	fmt.Fprintln(os.Stderr, "\nGlobal Functions:")
	for _, g := range api.GlobalFunctions {
		fmt.Fprintf(os.Stderr, "  %s\n", g.Name)
	}
}

func printDefines(defs []Define, indent string) {
	for _, d := range defs {
		if len(d.Values) > 0 {
			fmt.Fprintf(os.Stderr, "%s%s (%d values)\n", indent, d.Name, len(d.Values))
		} else if len(d.Subkeys) > 0 {
			fmt.Fprintf(os.Stderr, "%s%s\n", indent, d.Name)
			printDefines(d.Subkeys, indent+"  ")
		} else {
			fmt.Fprintf(os.Stderr, "%s%s\n", indent, d.Name)
		}
	}
}

// matchesFilter returns true if the named item should be included.
func matchesFilter(name string, allowed []string) bool {
	return slices.Contains(allowed, name)
}
