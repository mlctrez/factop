local tiles_mod = {}
local c = require("factop.common")

-- Tile manipulation module for factop softmod.
-- Provides functions to fill, read, remove, replace, and checkerboard tiles
-- over rectangular areas. Exposed via custom console commands for external
-- access through RCON.

-- ---------------------------------------------------------------------------
-- Core tile operations
-- ---------------------------------------------------------------------------

--- Fill a rectangular area with a single tile type.
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
-- Console commands
-- ---------------------------------------------------------------------------

local function register_commands()

    commands.add_command("tiles-fill", "Fill area with tiles. Usage: /tiles-fill x1,y1,x2,y2 tile_name [surface]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 2 then c.reply("Usage: /tiles-fill x1,y1,x2,y2 tile_name [surface]") return end
        local lt, rb = c.parse_tile_area(args[1])
        if not lt then c.reply("Invalid area format. Use: x1,y1,x2,y2") return end
        local surface = c.get_surface(args[3])
        if not surface then c.reply("Surface not found") return end
        c.reply("Placed " .. fill_area(surface, lt, rb, args[2], false) .. " tiles")
    end)

    commands.add_command("tiles-read", "Read tiles in area. Usage: /tiles-read x1,y1,x2,y2 [filter_name] [surface]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 1 then c.reply("Usage: /tiles-read x1,y1,x2,y2 [filter_name] [surface]") return end
        local lt, rb = c.parse_tile_area(args[1])
        if not lt then c.reply("Invalid area format. Use: x1,y1,x2,y2") return end
        local filter = args[2]
        if filter == "" then filter = nil end
        local surface = c.get_surface(args[3])
        if not surface then c.reply("Surface not found") return end
        local result = read_area(surface, lt, rb, filter)
        local parts = {}
        for _, t in ipairs(result) do
            parts[#parts + 1] = string.format("%s:%d:%d", t.name, t.x, t.y)
        end
        c.reply(table.concat(parts, ","))
    end)

    commands.add_command("tiles-remove", "Remove tiles in area (restore hidden). Usage: /tiles-remove x1,y1,x2,y2 [filter_name] [surface]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 1 then c.reply("Usage: /tiles-remove x1,y1,x2,y2 [filter_name] [surface]") return end
        local lt, rb = c.parse_tile_area(args[1])
        if not lt then c.reply("Invalid area format. Use: x1,y1,x2,y2") return end
        local filter = args[2]
        if filter == "" then filter = nil end
        local surface = c.get_surface(args[3])
        if not surface then c.reply("Surface not found") return end
        c.reply("Removed " .. remove_area(surface, lt, rb, filter) .. " tiles")
    end)

    commands.add_command("tiles-replace", "Replace tiles in area. Usage: /tiles-replace x1,y1,x2,y2 from_name to_name [surface]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 3 then c.reply("Usage: /tiles-replace x1,y1,x2,y2 from_name to_name [surface]") return end
        local lt, rb = c.parse_tile_area(args[1])
        if not lt then c.reply("Invalid area format. Use: x1,y1,x2,y2") return end
        local surface = c.get_surface(args[4])
        if not surface then c.reply("Surface not found") return end
        c.reply("Replaced " .. replace_area(surface, lt, rb, args[2], args[3]) .. " tiles")
    end)

    commands.add_command("tiles-checker", "Checkerboard pattern. Usage: /tiles-checker x1,y1,x2,y2 tile_a tile_b [surface]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 3 then c.reply("Usage: /tiles-checker x1,y1,x2,y2 tile_a tile_b [surface]") return end
        local lt, rb = c.parse_tile_area(args[1])
        if not lt then c.reply("Invalid area format. Use: x1,y1,x2,y2") return end
        local surface = c.get_surface(args[4])
        if not surface then c.reply("Surface not found") return end
        c.reply("Placed " .. checkerboard_area(surface, lt, rb, args[2], args[3]) .. " checkerboard tiles")
    end)

end

tiles_mod.on_init = register_commands
tiles_mod.on_load = register_commands

return tiles_mod
