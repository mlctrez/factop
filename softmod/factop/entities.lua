local entities_mod = {}

-- Entity manipulation module for factop softmod.
-- Provides basic CRUD operations for entities on a surface, exposed via
-- custom console commands for external access through RCON.

-- ---------------------------------------------------------------------------
-- Helpers
-- ---------------------------------------------------------------------------

--- Returns true if the command was invoked via RCON.
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

--- Parse "x1,y1,x2,y2" into a BoundingBox table.
local function parse_area(s)
    local x1, y1, x2, y2 = s:match("^([%-]?%d+%.?%d*),([%-]?%d+%.?%d*),([%-]?%d+%.?%d*),([%-]?%d+%.?%d*)$")
    if not x1 then return nil end
    x1, y1, x2, y2 = tonumber(x1), tonumber(y1), tonumber(x2), tonumber(y2)
    return { { math.min(x1, x2), math.min(y1, y2) }, { math.max(x1, x2), math.max(y1, y2) } }
end

local function get_surface(name)
    if name and name ~= "" then
        return game.surfaces[name]
    end
    return game.surfaces["nauvis"]
end

-- ---------------------------------------------------------------------------
-- Console commands
-- ---------------------------------------------------------------------------

local function register_commands()

    -- /entities-create <x,y> <name> [force] [direction] [surface]
    -- Creates a single entity at the given position.
    commands.add_command("entities-create", "Create entity. Usage: /entities-create x,y name [force] [direction] [surface]", function(cmd)
        if not rcon_only(cmd) then return end
        local args = {}
        if cmd.parameter then
            for w in cmd.parameter:gmatch("%S+") do args[#args + 1] = w end
        end
        if #args < 2 then
            reply("Usage: /entities-create x,y name [force] [direction] [surface]")
            return
        end
        local px, py = args[1]:match("^([%-]?%d+%.?%d*),([%-]?%d+%.?%d*)$")
        if not px then reply("Invalid position. Use: x,y") return end
        local pos = { x = tonumber(px), y = tonumber(py) }
        local name = args[2]
        local force = args[3] or "player"
        local direction = nil
        if args[4] then
            direction = defines.direction[args[4]]
        end
        local surface = get_surface(args[5])
        if not surface then reply("Surface not found") return end

        local params = { name = name, position = pos, force = force }
        if direction then params.direction = direction end

        local entity = surface.create_entity(params)
        if entity then
            reply(string.format("Created %s at {%.1f,%.1f} unit_number=%s",
                entity.name, entity.position.x, entity.position.y,
                tostring(entity.unit_number or "nil")))
        else
            reply("Failed to create entity")
        end
    end)

    -- /entities-find <x1,y1,x2,y2> [name] [type] [force] [limit] [surface]
    -- Finds entities in an area. Returns compact format: name:x:y:unit_number
    commands.add_command("entities-find", "Find entities. Usage: /entities-find x1,y1,x2,y2 [name] [type] [force] [limit] [surface]", function(cmd)
        if not rcon_only(cmd) then return end
        local args = {}
        if cmd.parameter then
            for w in cmd.parameter:gmatch("%S+") do args[#args + 1] = w end
        end
        if #args < 1 then
            reply("Usage: /entities-find x1,y1,x2,y2 [name] [type] [force] [limit] [surface]")
            return
        end
        local area = parse_area(args[1])
        if not area then reply("Invalid area format. Use: x1,y1,x2,y2") return end

        local filter = { area = area }
        if args[2] and args[2] ~= "" and args[2] ~= "_" then filter.name = args[2] end
        if args[3] and args[3] ~= "" and args[3] ~= "_" then filter.type = args[3] end
        if args[4] and args[4] ~= "" and args[4] ~= "_" then filter.force = args[4] end
        if args[5] and args[5] ~= "" and args[5] ~= "_" then filter.limit = tonumber(args[5]) end

        local surface = get_surface(args[6])
        if not surface then reply("Surface not found") return end

        local entities = surface.find_entities_filtered(filter)
        local parts = {}
        for _, e in ipairs(entities) do
            parts[#parts + 1] = string.format("%s:%.1f:%.1f:%s",
                e.name, e.position.x, e.position.y,
                tostring(e.unit_number or "0"))
        end
        reply(table.concat(parts, ","))
    end)

    -- /entities-count <x1,y1,x2,y2> [name] [type] [force] [surface]
    -- Counts entities matching the filter.
    commands.add_command("entities-count", "Count entities. Usage: /entities-count x1,y1,x2,y2 [name] [type] [force] [surface]", function(cmd)
        if not rcon_only(cmd) then return end
        local args = {}
        if cmd.parameter then
            for w in cmd.parameter:gmatch("%S+") do args[#args + 1] = w end
        end
        if #args < 1 then
            reply("Usage: /entities-count x1,y1,x2,y2 [name] [type] [force] [surface]")
            return
        end
        local area = parse_area(args[1])
        if not area then reply("Invalid area format. Use: x1,y1,x2,y2") return end

        local filter = { area = area }
        if args[2] and args[2] ~= "" and args[2] ~= "_" then filter.name = args[2] end
        if args[3] and args[3] ~= "" and args[3] ~= "_" then filter.type = args[3] end
        if args[4] and args[4] ~= "" and args[4] ~= "_" then filter.force = args[4] end

        local surface = get_surface(args[5])
        if not surface then reply("Surface not found") return end

        local count = surface.count_entities_filtered(filter)
        reply(tostring(count))
    end)

    -- /entities-destroy <x1,y1,x2,y2> [name] [type] [force] [limit] [surface]
    -- Destroys entities matching the filter in the area.
    commands.add_command("entities-destroy", "Destroy entities. Usage: /entities-destroy x1,y1,x2,y2 [name] [type] [force] [limit] [surface]", function(cmd)
        if not rcon_only(cmd) then return end
        local args = {}
        if cmd.parameter then
            for w in cmd.parameter:gmatch("%S+") do args[#args + 1] = w end
        end
        if #args < 1 then
            reply("Usage: /entities-destroy x1,y1,x2,y2 [name] [type] [force] [limit] [surface]")
            return
        end
        local area = parse_area(args[1])
        if not area then reply("Invalid area format. Use: x1,y1,x2,y2") return end

        local filter = { area = area }
        if args[2] and args[2] ~= "" and args[2] ~= "_" then filter.name = args[2] end
        if args[3] and args[3] ~= "" and args[3] ~= "_" then filter.type = args[3] end
        if args[4] and args[4] ~= "" and args[4] ~= "_" then filter.force = args[4] end
        if args[5] and args[5] ~= "" and args[5] ~= "_" then filter.limit = tonumber(args[5]) end

        local surface = get_surface(args[6])
        if not surface then reply("Surface not found") return end

        local entities = surface.find_entities_filtered(filter)
        local count = 0
        for _, e in ipairs(entities) do
            if e.valid and e.can_be_destroyed() then
                e.destroy({ raise_destroy = true })
                count = count + 1
            end
        end
        reply("Destroyed " .. count .. " entities")
    end)

end

-- ---------------------------------------------------------------------------
-- Event handler integration
-- ---------------------------------------------------------------------------

entities_mod.on_init = function()
    register_commands()
end

entities_mod.on_load = function()
    register_commands()
end

return entities_mod
