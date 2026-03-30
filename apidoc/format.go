package main

import (
	"fmt"
	"strings"
)

// formatType converts the polymorphic Type field into a readable string.
func formatType(t any) string {
	if t == nil {
		return "nil"
	}
	switch v := t.(type) {
	case string:
		return v
	case map[string]any:
		return formatComplexType(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func formatComplexType(m map[string]any) string {
	ct, _ := m["complex_type"].(string)
	switch ct {
	case "type":
		return formatType(m["value"])
	case "union":
		opts, _ := m["options"].([]any)
		parts := make([]string, len(opts))
		for i, o := range opts {
			parts[i] = formatType(o)
		}
		return strings.Join(parts, " | ")
	case "array":
		return "array[" + formatType(m["value"]) + "]"
	case "dictionary":
		return "dict[" + formatType(m["key"]) + " → " + formatType(m["value"]) + "]"
	case "LuaCustomTable":
		return "LuaCustomTable[" + formatType(m["key"]) + " → " + formatType(m["value"]) + "]"
	case "table":
		return "table"
	case "tuple":
		vals, _ := m["values"].([]any)
		parts := make([]string, len(vals))
		for i, v := range vals {
			parts[i] = formatType(v)
		}
		return "tuple[" + strings.Join(parts, ", ") + "]"
	case "function":
		return "function"
	case "literal":
		return fmt.Sprintf("%v", m["value"])
	case "LuaLazyLoadedValue":
		return "LuaLazyLoadedValue[" + formatType(m["value"]) + "]"
	case "LuaStruct":
		return "LuaStruct"
	default:
		return ct
	}
}

// formatParams builds a compact parameter signature.
func formatParams(params []Parameter, takesTable bool) string {
	if len(params) == 0 {
		return ""
	}
	if takesTable {
		return "{...}"
	}
	parts := make([]string, len(params))
	for i, p := range params {
		opt := ""
		if p.Optional {
			opt = "?"
		}
		parts[i] = p.Name + opt
	}
	return strings.Join(parts, ", ")
}

// trimDesc returns the first sentence of a description, cleaned up for inline use.
func trimDesc(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	// Truncate at first period followed by space for brevity
	if idx := strings.Index(s, ". "); idx > 0 && idx < 200 {
		return s[:idx+1]
	}
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}
