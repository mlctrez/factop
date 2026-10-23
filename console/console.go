// Package console validates plain Factorio console commands sent through
// factop's RCON console subject.
//
// Only commands with no Factorio Lua API equivalent are allowed. Everything
// else (players, admins, version, time, server-save, kick, ban, unban, mute,
// purge, promote/demote and so on) can be done with "/sc" Lua through
// factop.rcon, which also returns structured data via helpers.table_to_json.
package console

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Allowed lists the console commands that have no Lua equivalent in the
// Factorio 2.x runtime API (checked against 2.1.23), keyed by command name
// without the leading "/", with the reason each one is needed.
var Allowed = map[string]string{
	"whitelist": "the whitelist has no Lua API",
	"config":    "server settings such as password, max-players and visibility have no Lua API",
	"bans":      "listing banned players has no Lua API (ban/unban do: game.ban_player, game.unban_player)",
	"banlist":   "getting or clearing the ban list has no Lua API",
	"mutes":     "listing muted players has no Lua API (mute/unmute do: game.mute_player, game.unmute_player)",
}

// AllowedNames returns the allowed command names with a leading "/", sorted.
func AllowedNames() []string {
	names := make([]string, 0, len(Allowed))
	for name := range Allowed {
		names = append(names, "/"+name)
	}
	sort.Strings(names)
	return names
}

// Validate normalizes raw and checks that it is a single-line console command
// starting with "/" whose name is in Allowed.
func Validate(raw string) (string, error) {
	command := strings.TrimSpace(raw)
	switch {
	case command == "":
		return "", errors.New("console command is empty")
	case strings.ContainsAny(command, "\r\n"):
		return "", errors.New("console command must be a single line")
	case !strings.HasPrefix(command, "/"):
		return "", fmt.Errorf("console command must start with / (got %q); plain text would be sent as chat", command)
	}
	name := strings.TrimPrefix(strings.Fields(command)[0], "/")
	if _, ok := Allowed[name]; !ok {
		return "", fmt.Errorf("console command /%s is not allowed; only commands with no Lua equivalent are (%s). Use rcon with /sc Lua instead",
			name, strings.Join(AllowedNames(), ", "))
	}
	return command, nil
}
