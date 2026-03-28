# Soft mod files for factorio operator ( factop )

* https://github.com/mlctrez/factop/tree/master/softmod

## contents

* factop/*.lua - libraries passed to the built-in factorio event_handler code
* locale/<lang>/*.cfg - localization files
* img/* - any images bundled with the softmod
* softmod.go - golang code that creates the softmod zip payload
* controlHeader.lua - used by softmod.go to create the final control.lua

## Lua File Requirements (softmod/factop/*.lua)

All Lua files in the `softmod/factop/` directory are automatically included in the generated `control.lua` via `controlHeader.lua`. To interact correctly with the Factorio `event_handler`, each file must follow these rules:

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