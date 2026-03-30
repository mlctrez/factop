local surface_mod = {}

-- Surface properties and chunk management module for factop softmod.
-- Provides commands to query/modify surface-level state and manage chunks.

-- ---------------------------------------------------------------------------
-- Helpers
-- ---------------------------------------------------------------------------

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

local function get_surface(name)
    if name and name ~= "" then
        return game.surfaces[name]
    end
    return game.surfaces["nauvis"]
end

-- Writable boolean properties
local bool_props = {
    always_day = true, freeze_daytime = true, peaceful_mode = true,
    no_enemies_mode = true, show_clouds = true, generate_with_lab_tiles = true,
}

-- Writable numeric properties
local num_props = {
    daytime = true, wind_speed = true, wind_orientation = true,
    wind_orientation_change = true, solar_power_multiplier = true,
    min_brightness = true, ticks_per_day = true,
}

-- ---------------------------------------------------------------------------
-- Console commands
-- ---------------------------------------------------------------------------

local function register_commands()

    -- /surface-list
    commands.add_command("surface-list", "List all surfaces. Usage: /surface-list", function(cmd)
        if not rcon_only(cmd) then return end
        local parts = {}
        for _, s in pairs(game.surfaces) do
            parts[#parts + 1] = string.format("%s:%d", s.name, s.index)
        end
        reply(table.concat(parts, ","))
    end)

    -- /surface-info [name]
    commands.add_command("surface-info", "Get surface properties. Usage: /surface-info [name]", function(cmd)
        if not rcon_only(cmd) then return end
        local args = {}
        if cmd.parameter then
            for w in cmd.parameter:gmatch("%S+") do args[#args + 1] = w end
        end
        local surface = get_surface(args[1])
        if not surface then reply("Surface not found") return end

        local parts = {}
        parts[#parts + 1] = "name:" .. surface.name
        parts[#parts + 1] = "index:" .. surface.index
        parts[#parts + 1] = "always_day:" .. tostring(surface.always_day)
        parts[#parts + 1] = "daytime:" .. string.format("%.4f", surface.daytime)
        parts[#parts + 1] = "darkness:" .. string.format("%.4f", surface.darkness)
        parts[#parts + 1] = "freeze_daytime:" .. tostring(surface.freeze_daytime)
        parts[#parts + 1] = "peaceful_mode:" .. tostring(surface.peaceful_mode)
        parts[#parts + 1] = "no_enemies_mode:" .. tostring(surface.no_enemies_mode)
        parts[#parts + 1] = "wind_speed:" .. string.format("%.4f", surface.wind_speed)
        parts[#parts + 1] = "wind_orientation:" .. string.format("%.4f", surface.wind_orientation)
        parts[#parts + 1] = "solar_power_multiplier:" .. string.format("%.4f", surface.solar_power_multiplier)
        parts[#parts + 1] = "min_brightness:" .. string.format("%.4f", surface.min_brightness)
        parts[#parts + 1] = "ticks_per_day:" .. tostring(surface.ticks_per_day)
        parts[#parts + 1] = "show_clouds:" .. tostring(surface.show_clouds)
        parts[#parts + 1] = "generate_with_lab_tiles:" .. tostring(surface.generate_with_lab_tiles)
        reply(table.concat(parts, ","))
    end)

    -- /surface-set <property> <value> [name]
    commands.add_command("surface-set", "Set surface property. Usage: /surface-set property value [name]", function(cmd)
        if not rcon_only(cmd) then return end
        local args = {}
        if cmd.parameter then
            for w in cmd.parameter:gmatch("%S+") do args[#args + 1] = w end
        end
        if #args < 2 then
            reply("Usage: /surface-set property value [name]")
            return
        end
        local prop = args[1]
        local val = args[2]
        local surface = get_surface(args[3])
        if not surface then reply("Surface not found") return end

        if bool_props[prop] then
            if val == "true" then
                surface[prop] = true
            elseif val == "false" then
                surface[prop] = false
            else
                reply("Invalid boolean value: " .. val)
                return
            end
        elseif num_props[prop] then
            local n = tonumber(val)
            if not n then
                reply("Invalid number: " .. val)
                return
            end
            surface[prop] = n
        else
            reply("Unknown or read-only property: " .. prop)
            return
        end
        reply("Set " .. prop .. "=" .. val)
    end)

    -- /surface-generate <x,y> [radius] [surface]
    commands.add_command("surface-generate", "Generate chunks. Usage: /surface-generate x,y [radius] [surface]", function(cmd)
        if not rcon_only(cmd) then return end
        local args = {}
        if cmd.parameter then
            for w in cmd.parameter:gmatch("%S+") do args[#args + 1] = w end
        end
        if #args < 1 then
            reply("Usage: /surface-generate x,y [radius] [surface]")
            return
        end
        local px, py = args[1]:match("^([%-]?%d+%.?%d*),([%-]?%d+%.?%d*)$")
        if not px then reply("Invalid position. Use: x,y") return end
        local pos = { x = tonumber(px), y = tonumber(py) }
        local radius = tonumber(args[2]) or 0
        local surface = get_surface(args[3])
        if not surface then reply("Surface not found") return end

        surface.request_to_generate_chunks(pos, radius)
        surface.force_generate_chunk_requests()
        reply("Generated chunks at " .. args[1] .. " radius=" .. radius)
    end)

    -- /surface-delete-chunk <cx,cy> [surface]
    commands.add_command("surface-delete-chunk", "Delete a chunk. Usage: /surface-delete-chunk cx,cy [surface]", function(cmd)
        if not rcon_only(cmd) then return end
        local args = {}
        if cmd.parameter then
            for w in cmd.parameter:gmatch("%S+") do args[#args + 1] = w end
        end
        if #args < 1 then
            reply("Usage: /surface-delete-chunk cx,cy [surface]")
            return
        end
        local cx, cy = args[1]:match("^([%-]?%d+),([%-]?%d+)$")
        if not cx then reply("Invalid chunk position. Use: cx,cy") return end
        local surface = get_surface(args[2])
        if not surface then reply("Surface not found") return end

        surface.delete_chunk({ x = tonumber(cx), y = tonumber(cy) })
        reply("Deleted chunk " .. args[1])
    end)

    -- /surface-clear-pollution [surface]
    commands.add_command("surface-clear-pollution", "Clear all pollution. Usage: /surface-clear-pollution [surface]", function(cmd)
        if not rcon_only(cmd) then return end
        local args = {}
        if cmd.parameter then
            for w in cmd.parameter:gmatch("%S+") do args[#args + 1] = w end
        end
        local surface = get_surface(args[1])
        if not surface then reply("Surface not found") return end
        surface.clear_pollution()
        reply("Cleared pollution")
    end)

end

-- ---------------------------------------------------------------------------
-- Event handler integration
-- ---------------------------------------------------------------------------

surface_mod.on_init = function()
    register_commands()
end

surface_mod.on_load = function()
    register_commands()
end

return surface_mod
