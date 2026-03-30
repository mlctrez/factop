package main

import (
	"fmt"
	"strings"
)

func generateMarkdown(api *RuntimeAPI, filter Filter) string {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString("inclusion: manual\n")
	b.WriteString("---\n\n")
	b.WriteString(fmt.Sprintf("# Factorio Runtime Lua API Reference (v%s)\n\n", api.ApplicationVersion))

	if filter.isEmpty() {
		b.WriteString("No filters specified. Use `--list` to see available items, then pass filters.\n\n")
		b.WriteString("Example: `go run ./apidoc --classes LuaPlayer,LuaEntity --events on_player_created`\n")
		return b.String()
	}

	if filter.Globals {
		writeGlobals(&b, api)
	}
	writeClasses(&b, api, filter)
	writeEvents(&b, api, filter)
	writeDefines(&b, api, filter)
	writeConcepts(&b, api, filter)

	return b.String()
}
