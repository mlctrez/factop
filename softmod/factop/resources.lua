local resources_mod = {}

-- Resource and pollution manipulation module for factop softmod.
-- Provides commands to query resources, modify resource amounts, and
-- read/write pollution values.

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

local function parse_area(s)
    local x1, y1, x2, y2 = s:match("^([%-]?%d+%.?%d*),([%-]?%d+%.?%d*),([%-]?%d+%.?%d*),([%-]?%d+%.?%d*)$")
    if not x1 then return nil end
    x1, y1, x2, y2 = tonumber(x1), tonumber(y1), tonumber(x2), tonumber(y2)
    return { { math.min(x1, x2), math.min(y1, y2) }, { math.max(x1, x2), math.max(y1, y2) } }
end

-- ---------------------------------------------------------------------------
-- Console commands
-- ---------------------------------------------------------------------------

local function register_commands()

    -- /resources-count [surface]
    commands.add_command("resources-count", "Count all resources on surface. Usage: /resources-count [surface]", function(cmd)
        if not rcon_only(cmd) then return end
        local args = {}
        if cmd.parameter then
            for w in cmd.parameter:gmatch("%S+") do args[#args + 1] = w end
        end
        local surface = get_surface(args[1])
        if not surface then reply("Surface not found") return end

        local counts = surface.get_resource_counts()
        local parts = {}
        for name, count in pairs(counts) do
            parts[#parts + 1] = name .. ":" .. count
        end
        reply(table.concat(parts, ","))
    end)

    -- /resources-find <x1,y1,x2,y2> [name] [limit] [surface]
    commands.add_command("resources-find", "Find resources in area. Usage: /resources-find x1,y1,x2,y2 [name] [limit] [surface]", function(cmd)
        if not rcon_only(cmd) then return end
        local args = {}
        if cmd.parameter then
            for w in cmd.parameter:gmatch("%S+") do args[#args + 1] = w end
        end
        if #args < 1 then
            reply("Usage: /resources-find x1,y1,x2,y2 [name] [limit] [surface]")
            return
        end
        local area = parse_area(args[1])
        if not area then reply("Invalid area format. Use: x1,y1,x2,y2") return end

        local filter = { area = area, type = "resource" }
        if args[2] and args[2] ~= "" and args[2] ~= "_" then filter.name = args[2] end
        if args[3] and args[3] ~= "" and args[3] ~= "_" then filter.limit = tonumber(args[3]) end

        local surface = get_surface(args[4])
        if not surface then reply("Surface not found") return end

        local entities = surface.find_entities_filtered(filter)
        local parts = {}
        for _, e in ipairs(entities) do
            parts[#parts + 1] = string.format("%s:%.1f:%.1f:%d",
                e.name, e.position.x, e.position.y, e.amount)
        end
        reply(table.concat(parts, ","))
    end)

    -- /resources-set <x,y> <amount> [surface]
    commands.add_command("resources-set", "Set resource amount at position. Usage: /resources-set x,y amount [surface]", function(cmd)
        if not rcon_only(cmd) then return end
        local args = {}
        if cmd.parameter then
            for w in cmd.parameter:gmatch("%S+") do args[#args + 1] = w end
        end
        if #args < 2 then
            reply("Usage: /resources-set x,y amount [surface]")
            return
        end
        local px, py = args[1]:match("^([%-]?%d+%.?%d*),([%-]?%d+%.?%d*)$")
        if not px then reply("Invalid position. Use: x,y") return end
        local amount = tonumber(args[2])
        if not amount then reply("Invalid amount") return end

        local surface = get_surface(args[3])
        if not surface then reply("Surface not found") return end

        local entities = surface.find_entities_filtered({
            position = { x = tonumber(px), y = tonumber(py) },
            radius = 1,
            type = "resource",
            limit = 1,
        })
        if #entities == 0 then
            reply("No resource found at position")
            return
        end
        entities[1].amount = math.floor(amount)
        reply(string.format("Set %s amount=%d", entities[1].name, entities[1].amount))
    end)

    -- /pollution-get <x,y> [surface]
    commands.add_command("pollution-get", "Get pollution at position. Usage: /pollution-get x,y [surface]", function(cmd)
        if not rcon_only(cmd) then return end
        local args = {}
        if cmd.parameter then
            for w in cmd.parameter:gmatch("%S+") do args[#args + 1] = w end
        end
        if #args < 1 then
            reply("Usage: /pollution-get x,y [surface]")
            return
        end
        local px, py = args[1]:match("^([%-]?%d+%.?%d*),([%-]?%d+%.?%d*)$")
        if not px then reply("Invalid position. Use: x,y") return end

        local surface = get_surface(args[2])
        if not surface then reply("Surface not found") return end

        local val = surface.get_pollution({ x = tonumber(px), y = tonumber(py) })
        reply(string.format("%.2f", val))
    end)

    -- /pollution-set <x,y> <amount> [surface]
    commands.add_command("pollution-set", "Set pollution at position. Usage: /pollution-set x,y amount [surface]", function(cmd)
        if not rcon_only(cmd) then return end
        local args = {}
        if cmd.parameter then
            for w in cmd.parameter:gmatch("%S+") do args[#args + 1] = w end
        end
        if #args < 2 then
            reply("Usage: /pollution-set x,y amount [surface]")
            return
        end
        local px, py = args[1]:match("^([%-]?%d+%.?%d*),([%-]?%d+%.?%d*)$")
        if not px then reply("Invalid position. Use: x,y") return end
        local amount = tonumber(args[2])
        if not amount then reply("Invalid amount") return end

        local surface = get_surface(args[3])
        if not surface then reply("Surface not found") return end

        surface.set_pollution({ x = tonumber(px), y = tonumber(py) }, amount)
        reply(string.format("Set pollution=%.2f", amount))
    end)

    -- /pollution-total [surface]
    commands.add_command("pollution-total", "Get total pollution. Usage: /pollution-total [surface]", function(cmd)
        if not rcon_only(cmd) then return end
        local args = {}
        if cmd.parameter then
            for w in cmd.parameter:gmatch("%S+") do args[#args + 1] = w end
        end
        local surface = get_surface(args[1])
        if not surface then reply("Surface not found") return end

        reply(string.format("%.2f", surface.get_total_pollution()))
    end)

end

-- ---------------------------------------------------------------------------
-- Event handler integration
-- ---------------------------------------------------------------------------

resources_mod.on_init = function()
    register_commands()
end

resources_mod.on_load = function()
    register_commands()
end

return resources_mod
