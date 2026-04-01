local entity_mod = {}
local c = require("factop.common")

-- Entity manipulation module for factop softmod.
-- Provides basic CRUD operations for entities on a surface
-- and UDP event logging for entity lifecycle events.

local function log_entity_event(event_name, entity, player_index, extra)
    if not (entity and entity.valid) then return end
    local surface = entity.surface
    local pid = player_index or 0
    local msg = string.format("[%s] %s:%.1f:%.1f:%s:%s:%d:%d",
        event_name,
        entity.name, entity.position.x, entity.position.y,
        tostring(entity.unit_number or 0),
        surface.name, surface.index, pid)
    if extra then
        msg = msg .. ":" .. extra
    end
    if #msg > 512 then
        msg = msg:sub(1, 512)
    end
    helpers.send_udp(4000, msg, 0)
end

-- ---------------------------------------------------------------------------
-- Event handlers
-- ---------------------------------------------------------------------------

local function on_entity_died(event)
    local cause_name = "unknown"
    if event.cause and event.cause.valid then
        cause_name = event.cause.name
    end
    log_entity_event("entity-died", event.entity, 0, cause_name)
end

local function on_built_entity(event)
    log_entity_event("entity-built", event.entity, event.player_index)
end

local function on_player_mined_entity(event)
    log_entity_event("entity-mined", event.entity, event.player_index)
end

-- ---------------------------------------------------------------------------
-- Console commands
-- ---------------------------------------------------------------------------

local function register_commands()

    commands.add_command("entity-create", "Create entity. Usage: /entity-create x,y name [force] [direction] [surface]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 2 then c.reply("Usage: /entity-create x,y name [force] [direction] [surface]") return end
        local pos = c.parse_position(args[1])
        if not pos then c.reply("Invalid position. Use: x,y") return end
        local name = args[2]
        local force = args[3] or "player"
        local direction = nil
        if args[4] then direction = defines.direction[args[4]] end
        local surface = c.get_surface(args[5])
        if not surface then c.reply("Surface not found") return end

        local params = { name = name, position = pos, force = force }
        if direction then params.direction = direction end

        local entity = surface.create_entity(params)
        if entity then
            c.reply(string.format("Created %s at {%.1f,%.1f} unit_number=%s",
                entity.name, entity.position.x, entity.position.y,
                tostring(entity.unit_number or "nil")))
        else
            c.reply("Failed to create entity")
        end
    end)

    -- /entity-bulk <name> <force> <positions> [surface]
    -- Positions are semicolon-separated x,y pairs: x1,y1;x2,y2;x3,y3
    -- Creates multiple entities of the same type in a single RCON call.
    commands.add_command("entity-bulk", "Bulk create entities. Usage: /entity-bulk name force x1,y1;x2,y2;... [surface]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 3 then
            c.reply("Usage: /entity-bulk name force x1,y1;x2,y2;... [surface]")
            return
        end
        local name = args[1]
        local force = args[2]
        local surface = c.get_surface(args[4])
        if not surface then c.reply("Surface not found") return end

        local count = 0
        local failed = 0
        for pair in args[3]:gmatch("[^;]+") do
            local pos = c.parse_position(pair)
            if pos then
                local entity = surface.create_entity({ name = name, position = pos, force = force })
                if entity then
                    count = count + 1
                else
                    failed = failed + 1
                end
            else
                failed = failed + 1
            end
        end
        if failed > 0 then
            c.reply(string.format("Created %d, failed %d", count, failed))
        else
            c.reply(string.format("Created %d", count))
        end
    end)

    commands.add_command("entity-find", "Find entities. Usage: /entity-find x1,y1,x2,y2 [name] [type] [force] [limit] [surface]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 1 then c.reply("Usage: /entity-find x1,y1,x2,y2 [name] [type] [force] [limit] [surface]") return end
        local area = c.parse_area(args[1])
        if not area then c.reply("Invalid area format. Use: x1,y1,x2,y2") return end

        local filter = { area = area }
        if args[2] and args[2] ~= "" and args[2] ~= "_" then filter.name = args[2] end
        if args[3] and args[3] ~= "" and args[3] ~= "_" then filter.type = args[3] end
        if args[4] and args[4] ~= "" and args[4] ~= "_" then filter.force = args[4] end
        if args[5] and args[5] ~= "" and args[5] ~= "_" then filter.limit = tonumber(args[5]) end

        local surface = c.get_surface(args[6])
        if not surface then c.reply("Surface not found") return end

        local entities = surface.find_entities_filtered(filter)
        local parts = {}
        for _, e in ipairs(entities) do
            parts[#parts + 1] = string.format("%s:%.1f:%.1f:%s",
                e.name, e.position.x, e.position.y,
                tostring(e.unit_number or "0"))
        end
        c.reply(table.concat(parts, ","))
    end)

    commands.add_command("entity-count", "Count entities. Usage: /entity-count x1,y1,x2,y2 [name] [type] [force] [surface]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 1 then c.reply("Usage: /entity-count x1,y1,x2,y2 [name] [type] [force] [surface]") return end
        local area = c.parse_area(args[1])
        if not area then c.reply("Invalid area format. Use: x1,y1,x2,y2") return end

        local filter = { area = area }
        if args[2] and args[2] ~= "" and args[2] ~= "_" then filter.name = args[2] end
        if args[3] and args[3] ~= "" and args[3] ~= "_" then filter.type = args[3] end
        if args[4] and args[4] ~= "" and args[4] ~= "_" then filter.force = args[4] end

        local surface = c.get_surface(args[5])
        if not surface then c.reply("Surface not found") return end

        c.reply(tostring(surface.count_entities_filtered(filter)))
    end)

    commands.add_command("entity-destroy", "Destroy entities. Usage: /entity-destroy x1,y1,x2,y2 [name] [type] [force] [limit] [surface]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 1 then c.reply("Usage: /entity-destroy x1,y1,x2,y2 [name] [type] [force] [limit] [surface]") return end
        local area = c.parse_area(args[1])
        if not area then c.reply("Invalid area format. Use: x1,y1,x2,y2") return end

        local filter = { area = area }
        if args[2] and args[2] ~= "" and args[2] ~= "_" then filter.name = args[2] end
        if args[3] and args[3] ~= "" and args[3] ~= "_" then filter.type = args[3] end
        if args[4] and args[4] ~= "" and args[4] ~= "_" then filter.force = args[4] end
        if args[5] and args[5] ~= "" and args[5] ~= "_" then filter.limit = tonumber(args[5]) end

        local surface = c.get_surface(args[6])
        if not surface then c.reply("Surface not found") return end

        local entities = surface.find_entities_filtered(filter)
        local count = 0
        for _, e in ipairs(entities) do
            if e.valid and e.can_be_destroyed() then
                e.destroy({ raise_destroy = true })
                count = count + 1
            end
        end
        c.reply("Destroyed " .. count .. " entities")
    end)

end

entity_mod.events = {
    [defines.events.on_entity_died] = on_entity_died,
    [defines.events.on_built_entity] = on_built_entity,
    [defines.events.on_player_mined_entity] = on_player_mined_entity,
}

entity_mod.on_init = register_commands
entity_mod.on_load = register_commands

return entity_mod
