local tiles_mod = {}

-- Tile manipulation module for factop softmod.
-- Provides functions to fill, read, remove, replace, and checkerboard tiles
-- over rectangular areas. Exposed via custom console commands for external
-- access through RCON.

-- ---------------------------------------------------------------------------
-- Core tile operations
-- ---------------------------------------------------------------------------

--- Fill a rectangular area with a single tile type.
-- @param surface LuaSurface
-- @param left_top {x, y} top-left corner
-- @param right_bottom {x, y} bottom-right corner
-- @param tile_name string prototype name (e.g. "concrete")
-- @param raise boolean whether to raise script_raised_set_tiles
local function fill_area(surface, left_top, right_bottom, tile_name, raise)
    local tiles = {}
    for x = left_top.x, right_bottom.x - 1 do
        for y = left_top.y, right_bottom.y - 1 do
            tiles[#tiles + 1] = { name = tile_name, position = { x, y } }
        end
    end
    if #tiles > 0 then
        surface.set_tiles(tiles, true, raise or false)
    end
    return #tiles
end

--- Read tiles in a rectangular area, optionally filtered by name.
-- Returns a compact array of {name, x, y} tables.
-- @param surface LuaSurface
-- @param left_top {x, y}
-- @param right_bottom {x, y}
-- @param filter_name string|nil only include tiles matching this name
local function read_area(surface, left_top, right_bottom, filter_name)
    local result = {}
    for x = left_top.x, right_bottom.x - 1 do
        for y = left_top.y, right_bottom.y - 1 do
            local tile = surface.get_tile(x, y)
            if tile and tile.valid then
                if filter_name == nil or tile.name == filter_name then
                    result[#result + 1] = { name = tile.name, x = x, y = y }
                end
            end
        end
    end
    return result
end

--- Remove (revert) tiles in an area by restoring hidden tiles.
-- If a tile has a hidden_tile it is restored; otherwise the tile is replaced
-- with the surface's default tile.
-- @param surface LuaSurface
-- @param left_top {x, y}
-- @param right_bottom {x, y}
-- @param filter_name string|nil only remove tiles matching this name
local function remove_area(surface, left_top, right_bottom, filter_name)
    local tiles = {}
    for x = left_top.x, right_bottom.x - 1 do
        for y = left_top.y, right_bottom.y - 1 do
            local tile = surface.get_tile(x, y)
            if tile and tile.valid then
                if filter_name == nil or tile.name == filter_name then
                    local restore = tile.hidden_tile or "landfill"
                    tiles[#tiles + 1] = { name = restore, position = { x, y } }
                end
            end
        end
    end
    if #tiles > 0 then
        surface.set_tiles(tiles, true, false)
    end
    return #tiles
end

--- Replace one tile type with another in an area.
-- @param surface LuaSurface
-- @param left_top {x, y}
-- @param right_bottom {x, y}
-- @param from_name string tile name to match
-- @param to_name string tile name to place
local function replace_area(surface, left_top, right_bottom, from_name, to_name)
    local tiles = {}
    for x = left_top.x, right_bottom.x - 1 do
        for y = left_top.y, right_bottom.y - 1 do
            local tile = surface.get_tile(x, y)
            if tile and tile.valid and tile.name == from_name then
                tiles[#tiles + 1] = { name = to_name, position = { x, y } }
            end
        end
    end
    if #tiles > 0 then
        surface.set_tiles(tiles, true, false)
    end
    return #tiles
end

--- Fill an area with a checkerboard pattern of two tile types.
-- @param surface LuaSurface
-- @param left_top {x, y}
-- @param right_bottom {x, y}
-- @param tile_a string first tile name
-- @param tile_b string second tile name
local function checkerboard_area(surface, left_top, right_bottom, tile_a, tile_b)
    local tiles = {}
    for x = left_top.x, right_bottom.x - 1 do
        for y = left_top.y, right_bottom.y - 1 do
            local name = ((x + y) % 2 == 0) and tile_a or tile_b
            tiles[#tiles + 1] = { name = name, position = { x, y } }
        end
    end
    if #tiles > 0 then
        surface.set_tiles(tiles, true, false)
    end
    return #tiles
end

-- ---------------------------------------------------------------------------
-- Helpers
-- ---------------------------------------------------------------------------

--- Parse "x1,y1,x2,y2" into left_top and right_bottom tables.
-- Ensures left_top < right_bottom by sorting.
local function parse_area(s)
    local x1, y1, x2, y2 = s:match("^([%-]?%d+),([%-]?%d+),([%-]?%d+),([%-]?%d+)$")
    if not x1 then return nil end
    x1, y1, x2, y2 = tonumber(x1), tonumber(y1), tonumber(x2), tonumber(y2)
    return { x = math.min(x1, x2), y = math.min(y1, y2) },
           { x = math.max(x1, x2), y = math.max(y1, y2) }
end

--- Get the surface, defaulting to the first player's surface or nauvis.
local function get_surface(name)
    if name and name ~= "" then
        return game.surfaces[name]
    end
    return game.surfaces["nauvis"]
end

--- Returns true if the command was invoked via RCON (player_index is nil).
-- Rejects in-game player invocations with a message.
local function rcon_only(cmd)
    if cmd.player_index ~= nil then
        game.players[cmd.player_index].print("This command is only available via RCON")
        return false
    end
    return true
end

--- Print result to rcon if available, otherwise game.print.
local function reply(msg)
    if rcon and rcon.print then
        rcon.print(msg)
    else
        game.print(msg)
    end
end

-- ---------------------------------------------------------------------------
-- Console commands
-- ---------------------------------------------------------------------------
-- All commands use the format: /command <area> [args...]
-- where <area> is "x1,y1,x2,y2" (integer tile coordinates).
--
-- Commands are registered in on_init and on_load so they are available
-- after both new game creation and save loading.

local function register_commands()

    -- /tiles-fill <x1,y1,x2,y2> <tile_name> [surface]
    commands.add_command("tiles-fill", "Fill area with tiles. Usage: /tiles-fill x1,y1,x2,y2 tile_name [surface]", function(cmd)
        if not rcon_only(cmd) then return end
        local args = {}
        if cmd.parameter then
            for w in cmd.parameter:gmatch("%S+") do args[#args + 1] = w end
        end
        if #args < 2 then
            reply("Usage: /tiles-fill x1,y1,x2,y2 tile_name [surface]")
            return
        end
        local lt, rb = parse_area(args[1])
        if not lt then reply("Invalid area format. Use: x1,y1,x2,y2") return end
        local surface = get_surface(args[3])
        if not surface then reply("Surface not found") return end
        local count = fill_area(surface, lt, rb, args[2], false)
        reply("Placed " .. count .. " tiles")
    end)

    -- /tiles-read <x1,y1,x2,y2> [filter_name] [surface]
    commands.add_command("tiles-read", "Read tiles in area. Usage: /tiles-read x1,y1,x2,y2 [filter_name] [surface]", function(cmd)
        if not rcon_only(cmd) then return end
        local args = {}
        if cmd.parameter then
            for w in cmd.parameter:gmatch("%S+") do args[#args + 1] = w end
        end
        if #args < 1 then
            reply("Usage: /tiles-read x1,y1,x2,y2 [filter_name] [surface]")
            return
        end
        local lt, rb = parse_area(args[1])
        if not lt then reply("Invalid area format. Use: x1,y1,x2,y2") return end
        local filter = args[2]
        if filter == "" then filter = nil end
        local surface = get_surface(args[3])
        if not surface then reply("Surface not found") return end
        local result = read_area(surface, lt, rb, filter)
        -- Return compact JSON-like output for external consumption
        local parts = {}
        for _, t in ipairs(result) do
            parts[#parts + 1] = string.format("%s:%d:%d", t.name, t.x, t.y)
        end
        reply(table.concat(parts, ","))
    end)

    -- /tiles-remove <x1,y1,x2,y2> [filter_name] [surface]
    commands.add_command("tiles-remove", "Remove tiles in area (restore hidden). Usage: /tiles-remove x1,y1,x2,y2 [filter_name] [surface]", function(cmd)
        if not rcon_only(cmd) then return end
        local args = {}
        if cmd.parameter then
            for w in cmd.parameter:gmatch("%S+") do args[#args + 1] = w end
        end
        if #args < 1 then
            reply("Usage: /tiles-remove x1,y1,x2,y2 [filter_name] [surface]")
            return
        end
        local lt, rb = parse_area(args[1])
        if not lt then reply("Invalid area format. Use: x1,y1,x2,y2") return end
        local filter = args[2]
        if filter == "" then filter = nil end
        local surface = get_surface(args[3])
        if not surface then reply("Surface not found") return end
        local count = remove_area(surface, lt, rb, filter)
        reply("Removed " .. count .. " tiles")
    end)

    -- /tiles-replace <x1,y1,x2,y2> <from_name> <to_name> [surface]
    commands.add_command("tiles-replace", "Replace tiles in area. Usage: /tiles-replace x1,y1,x2,y2 from_name to_name [surface]", function(cmd)
        if not rcon_only(cmd) then return end
        local args = {}
        if cmd.parameter then
            for w in cmd.parameter:gmatch("%S+") do args[#args + 1] = w end
        end
        if #args < 3 then
            reply("Usage: /tiles-replace x1,y1,x2,y2 from_name to_name [surface]")
            return
        end
        local lt, rb = parse_area(args[1])
        if not lt then reply("Invalid area format. Use: x1,y1,x2,y2") return end
        local surface = get_surface(args[3 + 1])
        if not surface then reply("Surface not found") return end
        local count = replace_area(surface, lt, rb, args[2], args[3])
        reply("Replaced " .. count .. " tiles")
    end)

    -- /tiles-checker <x1,y1,x2,y2> <tile_a> <tile_b> [surface]
    commands.add_command("tiles-checker", "Checkerboard pattern. Usage: /tiles-checker x1,y1,x2,y2 tile_a tile_b [surface]", function(cmd)
        if not rcon_only(cmd) then return end
        local args = {}
        if cmd.parameter then
            for w in cmd.parameter:gmatch("%S+") do args[#args + 1] = w end
        end
        if #args < 3 then
            reply("Usage: /tiles-checker x1,y1,x2,y2 tile_a tile_b [surface]")
            return
        end
        local lt, rb = parse_area(args[1])
        if not lt then reply("Invalid area format. Use: x1,y1,x2,y2") return end
        local surface = get_surface(args[4])
        if not surface then reply("Surface not found") return end
        local count = checkerboard_area(surface, lt, rb, args[2], args[3])
        reply("Placed " .. count .. " checkerboard tiles")
    end)

end

-- ---------------------------------------------------------------------------
-- Event handler integration
-- ---------------------------------------------------------------------------

tiles_mod.on_init = function()
    register_commands()
end

tiles_mod.on_load = function()
    register_commands()
end

return tiles_mod
