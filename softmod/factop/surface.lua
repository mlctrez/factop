local surface_mod = {}
local c = require("factop.common")

-- Surface properties and chunk management module for factop softmod.

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

    commands.add_command("surface-list", "List all surfaces. Usage: /surface-list", function(cmd)
        if not c.rcon_only(cmd) then return end
        local parts = {}
        for _, s in pairs(game.surfaces) do
            parts[#parts + 1] = string.format("%s:%d", s.name, s.index)
        end
        c.reply(table.concat(parts, ","))
    end)

    commands.add_command("surface-info", "Get surface properties. Usage: /surface-info [name]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        local surface = c.get_surface(args[1])
        if not surface then c.reply("Surface not found") return end

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
        c.reply(table.concat(parts, ","))
    end)

    commands.add_command("surface-set", "Set surface property. Usage: /surface-set property value [name]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 2 then c.reply("Usage: /surface-set property value [name]") return end
        local prop = args[1]
        local val = args[2]
        local surface = c.get_surface(args[3])
        if not surface then c.reply("Surface not found") return end

        if bool_props[prop] then
            if val == "true" then surface[prop] = true
            elseif val == "false" then surface[prop] = false
            else c.reply("Invalid boolean value: " .. val) return end
        elseif num_props[prop] then
            local n = tonumber(val)
            if not n then c.reply("Invalid number: " .. val) return end
            surface[prop] = n
        else
            c.reply("Unknown or read-only property: " .. prop)
            return
        end
        c.reply("Set " .. prop .. "=" .. val)
    end)

    commands.add_command("surface-generate", "Generate chunks. Usage: /surface-generate x,y [radius] [surface]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 1 then c.reply("Usage: /surface-generate x,y [radius] [surface]") return end
        local pos = c.parse_position(args[1])
        if not pos then c.reply("Invalid position. Use: x,y") return end
        local radius = tonumber(args[2]) or 0
        local surface = c.get_surface(args[3])
        if not surface then c.reply("Surface not found") return end

        surface.request_to_generate_chunks(pos, radius)
        surface.force_generate_chunk_requests()
        c.reply("Generated chunks at " .. args[1] .. " radius=" .. radius)
    end)

    commands.add_command("surface-delete-chunk", "Delete a chunk. Usage: /surface-delete-chunk cx,cy [surface]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 1 then c.reply("Usage: /surface-delete-chunk cx,cy [surface]") return end
        local cx, cy = args[1]:match("^([%-]?%d+),([%-]?%d+)$")
        if not cx then c.reply("Invalid chunk position. Use: cx,cy") return end
        local surface = c.get_surface(args[2])
        if not surface then c.reply("Surface not found") return end
        surface.delete_chunk({ x = tonumber(cx), y = tonumber(cy) })
        c.reply("Deleted chunk " .. args[1])
    end)

    commands.add_command("surface-clear-pollution", "Clear all pollution. Usage: /surface-clear-pollution [surface]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        local surface = c.get_surface(args[1])
        if not surface then c.reply("Surface not found") return end
        surface.clear_pollution()
        c.reply("Cleared pollution")
    end)

end

surface_mod.on_init = register_commands
surface_mod.on_load = register_commands

return surface_mod
