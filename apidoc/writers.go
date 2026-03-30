package main

import (
	"fmt"
	"strings"
)

func writeGlobals(b *strings.Builder, api *RuntimeAPI) {
	b.WriteString("## Global Objects\n\n")
	for _, g := range api.GlobalObjects {
		b.WriteString(fmt.Sprintf("- `%s` :: `%s`", g.Name, formatType(g.Type)))
		if g.Description != "" {
			b.WriteString(" — " + trimDesc(g.Description))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if len(api.GlobalFunctions) > 0 {
		b.WriteString("## Global Functions\n\n")
		for _, f := range api.GlobalFunctions {
			writeMethod(b, f, "")
		}
		b.WriteString("\n")
	}
}

func writeClasses(b *strings.Builder, api *RuntimeAPI, filter Filter) {
	if len(filter.Classes) == 0 {
		return
	}
	for _, c := range api.Classes {
		if !matchesFilter(c.Name, filter.Classes) {
			continue
		}
		writeClass(b, c)
	}
}

func writeClass(b *strings.Builder, c Class) {
	header := fmt.Sprintf("## Class: %s", c.Name)
	if c.Parent != "" {
		header += fmt.Sprintf(" (extends %s)", c.Parent)
	}
	if c.Abstract {
		header += " [abstract]"
	}
	b.WriteString(header + "\n\n")

	if c.Description != "" {
		b.WriteString(trimDesc(c.Description) + "\n\n")
	}

	if len(c.Attributes) > 0 {
		b.WriteString("### Attributes\n\n")
		for _, a := range c.Attributes {
			writeAttribute(b, a)
		}
		b.WriteString("\n")
	}

	if len(c.Methods) > 0 {
		b.WriteString("### Methods\n\n")
		for _, m := range c.Methods {
			writeMethod(b, m, c.Name)
		}
		b.WriteString("\n")
	}
}

func writeAttribute(b *strings.Builder, a Attribute) {
	rw := ""
	if a.ReadType != nil && a.WriteType != nil {
		rw = "RW"
	} else if a.ReadType != nil {
		rw = "R"
	} else if a.WriteType != nil {
		rw = "W"
	}

	typ := formatType(a.ReadType)
	if typ == "" {
		typ = formatType(a.WriteType)
	}

	opt := ""
	if a.Optional {
		opt = "?"
	}

	b.WriteString(fmt.Sprintf("- `%s%s` :: `%s` [%s]", a.Name, opt, typ, rw))
	if a.Description != "" {
		b.WriteString(" — " + trimDesc(a.Description))
	}
	b.WriteString("\n")
}

func writeMethod(b *strings.Builder, m Method, className string) {
	prefix := ""
	if className != "" {
		prefix = className + "."
	}

	params := formatParams(m.Parameters, m.Format.TakesTable)
	ret := ""
	if len(m.ReturnValues) > 0 {
		types := make([]string, len(m.ReturnValues))
		for i, r := range m.ReturnValues {
			types[i] = formatType(r.Type)
		}
		ret = " → " + strings.Join(types, ", ")
	}

	b.WriteString(fmt.Sprintf("- `%s%s(%s)%s`", prefix, m.Name, params, ret))
	if m.Description != "" {
		b.WriteString(" — " + trimDesc(m.Description))
	}
	b.WriteString("\n")

	// Write parameter details for table-style methods
	if m.Format.TakesTable && len(m.Parameters) > 0 {
		for _, p := range m.Parameters {
			opt := ""
			if p.Optional {
				opt = "?"
			}
			b.WriteString(fmt.Sprintf("  - `%s%s` :: `%s`", p.Name, opt, formatType(p.Type)))
			if p.Description != "" {
				b.WriteString(" — " + trimDesc(p.Description))
			}
			b.WriteString("\n")
		}
	}
}

func writeEvents(b *strings.Builder, api *RuntimeAPI, filter Filter) {
	if len(filter.Events) == 0 {
		return
	}
	first := true
	for _, e := range api.Events {
		if !matchesFilter(e.Name, filter.Events) {
			continue
		}
		if first {
			b.WriteString("## Events\n\n")
			first = false
		}
		b.WriteString(fmt.Sprintf("### %s\n\n", e.Name))
		if e.Description != "" {
			b.WriteString(trimDesc(e.Description) + "\n\n")
		}
		if len(e.Data) > 0 {
			b.WriteString("Fields:\n")
			for _, d := range e.Data {
				opt := ""
				if d.Optional {
					opt = "?"
				}
				b.WriteString(fmt.Sprintf("- `%s%s` :: `%s`", d.Name, opt, formatType(d.Type)))
				if d.Description != "" {
					b.WriteString(" — " + trimDesc(d.Description))
				}
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}
	}
}

func writeDefines(b *strings.Builder, api *RuntimeAPI, filter Filter) {
	if len(filter.Defines) == 0 {
		return
	}
	first := true
	for _, d := range api.Defines {
		if !matchesFilter(d.Name, filter.Defines) {
			continue
		}
		if first {
			b.WriteString("## Defines\n\n")
			first = false
		}
		writeDefine(b, d, "defines")
	}
}

func writeDefine(b *strings.Builder, d Define, prefix string) {
	fullName := prefix + "." + d.Name
	if len(d.Values) > 0 {
		b.WriteString(fmt.Sprintf("### %s\n\n", fullName))
		if d.Description != "" {
			b.WriteString(trimDesc(d.Description) + "\n\n")
		}
		for _, v := range d.Values {
			b.WriteString(fmt.Sprintf("- `%s.%s`", fullName, v.Name))
			if v.Description != "" {
				b.WriteString(" — " + trimDesc(v.Description))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	for _, sub := range d.Subkeys {
		writeDefine(b, sub, fullName)
	}
}

func writeConcepts(b *strings.Builder, api *RuntimeAPI, filter Filter) {
	if len(filter.Concepts) == 0 {
		return
	}
	first := true
	for _, c := range api.Concepts {
		if !matchesFilter(c.Name, filter.Concepts) {
			continue
		}
		if first {
			b.WriteString("## Concepts\n\n")
			first = false
		}
		b.WriteString(fmt.Sprintf("### %s\n\n", c.Name))
		if c.Description != "" {
			b.WriteString(trimDesc(c.Description) + "\n\n")
		}
		b.WriteString(fmt.Sprintf("Type: `%s`\n\n", formatType(c.Type)))
	}
}
