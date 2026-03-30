package main

import (
	"encoding/json"
	"sort"
	"strings"
	"unicode"
)

// defaultTypes are the prototype categories most relevant for entity/tile
// manipulation via the softmod commands.
var defaultTypes = []string{
	// Tiles
	"tile",
	// Placeable entities (the types you'd use with create_entity)
	"accumulator", "ammo-turret", "arithmetic-combinator", "artillery-turret",
	"artillery-wagon", "assembling-machine", "asteroid-collector",
	"beacon", "boiler", "burner-generator",
	"car", "cargo-bay", "cargo-landing-pad", "cargo-wagon",
	"character", "cliff", "combat-robot", "constant-combinator",
	"construction-robot", "container",
	"curved-rail-a", "curved-rail-b",
	"decider-combinator", "display-panel",
	"electric-energy-interface", "electric-pole", "electric-turret",
	"elevated-curved-rail-a", "elevated-curved-rail-b",
	"elevated-half-diagonal-rail", "elevated-straight-rail",
	"fluid-turret", "fluid-wagon", "furnace",
	"gate", "generator",
	"half-diagonal-rail", "heat-interface", "heat-pipe",
	"infinity-container", "infinity-pipe", "inserter",
	"lab", "lamp", "land-mine", "lane-splitter",
	"linked-belt", "linked-container", "loader", "loader-1x1",
	"locomotive", "logistic-container", "logistic-robot",
	"mining-drill",
	"offshore-pump",
	"pipe", "pipe-to-ground", "power-switch", "programmable-speaker", "pump",
	"radar", "rail-chain-signal", "rail-ramp", "rail-signal",
	"rail-support", "reactor", "roboport", "rocket-silo",
	"selector-combinator", "simple-entity", "simple-entity-with-force",
	"simple-entity-with-owner", "solar-panel", "spider-vehicle",
	"splitter", "storage-tank", "straight-rail",
	"train-stop", "transport-belt",
	"tree", "turret",
	"underground-belt",
	"wall",
	// Resources
	"resource",
}

// ProtoGroup holds all prototype names for a single type.
type ProtoGroup struct {
	Type  string
	GoID  string // PascalCase identifier for Go
	Names []string
}

func buildTypeFilter(csv string) map[string]bool {
	if csv == "" {
		m := make(map[string]bool, len(defaultTypes))
		for _, t := range defaultTypes {
			m[t] = true
		}
		return m
	}
	parts := strings.Split(csv, ",")
	m := make(map[string]bool, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			m[p] = true
		}
	}
	return m
}

func extractGroups(raw map[string]map[string]json.RawMessage, filter map[string]bool) []ProtoGroup {
	var groups []ProtoGroup
	for typeName, items := range raw {
		if !filter[typeName] {
			continue
		}
		names := make([]string, 0, len(items))
		for name := range items {
			names = append(names, name)
		}
		sort.Strings(names)
		groups = append(groups, ProtoGroup{
			Type:  typeName,
			GoID:  toGoID(typeName),
			Names: names,
		})
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Type < groups[j].Type
	})
	return groups
}

// toGoID converts "ammo-turret" to "AmmoTurret".
func toGoID(s string) string {
	var b strings.Builder
	upper := true
	for _, r := range s {
		if r == '-' || r == '_' {
			upper = true
			continue
		}
		if upper {
			b.WriteRune(unicode.ToUpper(r))
			upper = false
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
