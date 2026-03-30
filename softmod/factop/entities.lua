local entities_mod = {}
local c = require("factop.common")

-- Entity manipulation module for factop softmod.
-- Provides basic CRUD operations for entities on a surface.

-- ---------------------------------------------------------------------------
-- Console commands
-- ---------------------------------------------------------------------------

local function register_commands()

    commands.add_command("entities-create", "Create entity. Usage: /entities-create x,y name [force] [direction] [surface]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 2 then c.reply("Usage: /entities-create x,y name [force] [direction] [surface]") return end
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

    commands.add_command("entities-find", "Find entities. Usage: /entities-find x1,y1,x2,y2 [name] [type] [force] [limit] [surface]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 1 then c.reply("Usage: /entities-find x1,y1,x2,y2 [name] [type] [force] [limit] [surface]") return end
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

    commands.add_command("entities-count", "Count entities. Usage: /entities-count x1,y1,x2,y2 [name] [type] [force] [surface]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 1 then c.reply("Usage: /entities-count x1,y1,x2,y2 [name] [type] [force] [surface]") return end
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

    commands.add_command("entities-destroy", "Destroy entities. Usage: /entities-destroy x1,y1,x2,y2 [name] [type] [force] [limit] [surface]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 1 then c.reply("Usage: /entities-destroy x1,y1,x2,y2 [name] [type] [force] [limit] [surface]") return end
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

entities_mod.on_init = register_commands
entities_mod.on_load = register_commands

return entities_mod
