# Soft mod files for factorio operator ( factop )

* https://github.com/mlctrez/factop/tree/master/softmod

## contents

* factop/common.lua - shared helper functions (not registered with event_handler)
* factop/*.lua - libraries passed to the built-in factorio event_handler code
* locale/<lang>/*.cfg - localization files
* img/* - any images bundled with the softmod
* softmod.go - golang code that creates the softmod zip payload
* controlHeader.lua - used by softmod.go to create the final control.lua

## Lua File Requirements (softmod/factop/*.lua)

### Naming Convention

All Lua module files use singular nouns: `entity.lua`, `tile.lua`, `surface.lua`, `resource.lua`, `player.lua`. Do not use plural forms. The corresponding Go client package under `client/` must use the same singular name: `client/entity/`, `client/tile/`, etc.

All Lua files in the `softmod/factop/` directory are automatically included in the generated `control.lua` via `controlHeader.lua`, with one exception: `common.lua` is excluded from `add_lib` registration since it has no event handlers. It is instead required directly by modules that need it.

### Shared Helpers (factop/common.lua)

`common.lua` provides functions shared across all command modules. Require it at the top of any module:

```lua
local c = require("factop.common")
```

Available functions:
* `c.rcon_only(cmd)` — returns true if invoked via RCON, rejects in-game players
* `c.reply(msg)` — prints to rcon.print or game.print as appropriate
* `c.get_surface(name)` — returns the named surface or nauvis as default
* `c.parse_args(cmd)` — splits cmd.parameter into an array of strings
* `c.parse_tile_area(s)` — parses "x1,y1,x2,y2" into left_top/right_bottom (integer coords)
* `c.parse_area(s)` — parses "x1,y1,x2,y2" into a BoundingBox (float coords)
* `c.parse_position(s)` — parses "x,y" into a position table

### Module Requirements

To interact correctly with the Factorio `event_handler`, each file must follow these rules:

1. **Return a Table**: Each file MUST return a table containing the event handlers it wants to register.
2. **Event Registration**: The table should use the standard Factorio `event_handler` structure, including keys such as:
    * `events`: A dictionary of event IDs to handler functions.
    * `on_nth_tick`: A dictionary of tick intervals to handler functions.
    * `on_init`: A function to run when the mod is first initialized.
    * `on_load`: A function to run when the save is loaded.
    * `on_configuration_changed`: A function to run when mod configurations change.
3. **Log Message Format**: When logging player-related events via UDP, the message MUST include both the player name and the player index (ID).
    * **Recommended Format**: To minimize UDP packet size and ensure reliable delivery, use a compact format.
    * **Example**: `[event] #ID name data` (e.g., `[join] #1 user_name`)
    * **Packet Size**: UDP packets should be kept under 512 bytes to avoid fragmentation and ensure they are processed correctly by the `UDPBridge`.

### Example File (softmod/factop/example.lua)

```lua
local example = {}

example.on_player_created = function(event)
    game.print("Welcome to factop, " .. game.players[event.player_index].name)
end

example.events = {
    [defines.events.on_player_created] = example.on_player_created
}

return example
```

The `softmod.go` script iterates through all `.lua` files in `softmod/factop/`, converts their path to a require string (e.g., `factop.example`), and adds `add_lib("factop.example")` to the `control.lua`. The `add_lib` function in `controlHeader.lua` then requires the file and passes the returned table to `handler.add_lib(l)`.

## Registering Custom Console Commands

Modules can register custom console commands using the `commands` global (`LuaCommandProcessor`). This makes functions accessible externally via RCON without needing to wrap calls in `/sc` Lua snippets.

### Key rules for command registration:

1. **Register in both `on_init` and `on_load`**: `on_load` runs when an existing save is loaded — this is the path used during normal softmod deployment (the server is stopped, the save zip is patched, and the server restarts). `on_init` only runs when a brand new save is created (e.g. during a `reset` command). For softmod updates, `on_load` is the critical callback. However, both should be implemented so commands work regardless of how the save was started.
2. **Do not register in `events`**: Command registration is not event-driven. It happens during initialization only.
3. **RCON-only by default**: Commands registered by softmod modules are operational commands, not player-facing features. Every command handler MUST guard against in-game player invocation by checking `cmd.player_index`. When `player_index` is `nil`, the command was invoked via RCON. When it is set, a player typed it in the console — reject it.
4. **Use `rcon.print()` for output**: When a command is invoked via RCON, `game.print()` sends output to the in-game console but not back to the RCON caller. Use `rcon.print()` to return data to the external caller. The `rcon` global is only available during RCON execution, so check for it before use.
5. **Command handler signature**: The handler receives a single table with `name` (command name), `player_index` (nil when called via RCON), and `parameter` (the string after the command name).
6. **Avoid duplicate registration**: `commands.add_command` will error if a command with the same name is already registered. Since `on_load` is called on every save load, and `on_init` on first creation, the registration function should be idempotent or called from a shared helper.

### Example: command registration pattern

```lua
local my_mod = {}

--- Returns true if the command was invoked via RCON (player_index is nil).
-- Rejects in-game player invocations with a message.
local function rcon_only(cmd)
    if cmd.player_index ~= nil then
        game.players[cmd.player_index].print("This command is only available via RCON")
        return false
    end
    return true
end

local function reply(msg)
    if rcon and rcon.print then
        rcon.print(msg)
    else
        game.print(msg)
    end
end

local function register_commands()
    commands.add_command("my-cmd", "Description here", function(cmd)
        if not rcon_only(cmd) then return end
        -- cmd.parameter is the string after "/my-cmd "
        local args = {}
        if cmd.parameter then
            for w in cmd.parameter:gmatch("%S+") do args[#args + 1] = w end
        end
        reply("result: " .. args[1])
    end)
end

my_mod.on_init = register_commands
my_mod.on_load = register_commands

return my_mod
```

External invocation via RCON (through NATS):
```
/tile-fill -10,-10,10,10 concrete
```

## Tile Manipulation (factop/tiles.lua)

The `tiles.lua` module provides area-based tile operations exposed as console commands. All commands take an area as `x1,y1,x2,y2` (integer tile coordinates) and an optional surface name (defaults to `nauvis`).

### Available commands

| Command | Usage | Description |
|---------|-------|-------------|
| `/tile-fill` | `x1,y1,x2,y2 tile_name [surface]` | Fill rectangular area with a tile type |
| `/tile-read` | `x1,y1,x2,y2 [filter_name] [surface]` | Read tile names/positions in area. Returns compact `name:x:y,...` format |
| `/tile-remove` | `x1,y1,x2,y2 [filter_name] [surface]` | Restore hidden tiles in area (reverts placed tiles) |
| `/tile-replace` | `x1,y1,x2,y2 from_name to_name [surface]` | Replace one tile type with another |
| `/tile-checker` | `x1,y1,x2,y2 tile_a tile_b [surface]` | Fill area with alternating checkerboard pattern |

### Examples via RCON

```
/tile-fill -20,-20,20,20 concrete
/tile-read -5,-5,5,5
/tile-read -5,-5,5,5 concrete
/tile-remove -20,-20,20,20 concrete
/tile-replace -10,-10,10,10 grass-1 concrete
/tile-checker 0,0,16,16 concrete stone-path
```

### Design notes

- **Small payload, large result**: Commands take a bounding box and tile name — the loop to create individual tile entries happens server-side in Lua, keeping RCON payloads minimal.
- **`set_tiles` with `correct_tiles=true`**: Ensures tile transitions render correctly.
- **Remove restores hidden tiles**: When Factorio places tiles like concrete over grass, the original tile is stored as a "hidden tile". The remove operation restores it. Falls back to `landfill` if no hidden tile exists.
- **Read output format**: Uses `name:x:y` comma-separated format for compact, parseable output suitable for external tooling.