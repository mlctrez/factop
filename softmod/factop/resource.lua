local resource_mod = {}
local c = require("factop.common")

-- Resource and pollution manipulation module for factop softmod.

-- ---------------------------------------------------------------------------
-- Console commands
-- ---------------------------------------------------------------------------

local function register_commands()

    commands.add_command("resource-count", "Count all resources on surface. Usage: /resource-count [surface]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        local surface = c.get_surface(args[1])
        if not surface then c.reply("Surface not found") return end
        local counts = surface.get_resource_counts()
        local parts = {}
        for name, count in pairs(counts) do
            parts[#parts + 1] = name .. ":" .. count
        end
        c.reply(table.concat(parts, ","))
    end)

    commands.add_command("resource-find", "Find resources in area. Usage: /resource-find x1,y1,x2,y2 [name] [limit] [surface]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 1 then c.reply("Usage: /resource-find x1,y1,x2,y2 [name] [limit] [surface]") return end
        local area = c.parse_area(args[1])
        if not area then c.reply("Invalid area format. Use: x1,y1,x2,y2") return end

        local filter = { area = area, type = "resource" }
        if args[2] and args[2] ~= "" and args[2] ~= "_" then filter.name = args[2] end
        if args[3] and args[3] ~= "" and args[3] ~= "_" then filter.limit = tonumber(args[3]) end

        local surface = c.get_surface(args[4])
        if not surface then c.reply("Surface not found") return end

        local entities = surface.find_entities_filtered(filter)
        local parts = {}
        for _, e in ipairs(entities) do
            parts[#parts + 1] = string.format("%s:%.1f:%.1f:%d",
                e.name, e.position.x, e.position.y, e.amount)
        end
        c.reply(table.concat(parts, ","))
    end)

    commands.add_command("resource-set", "Set resource amount at position. Usage: /resource-set x,y amount [surface]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 2 then c.reply("Usage: /resource-set x,y amount [surface]") return end
        local pos = c.parse_position(args[1])
        if not pos then c.reply("Invalid position. Use: x,y") return end
        local amount = tonumber(args[2])
        if not amount then c.reply("Invalid amount") return end

        local surface = c.get_surface(args[3])
        if not surface then c.reply("Surface not found") return end

        local entities = surface.find_entities_filtered({
            position = pos, radius = 1, type = "resource", limit = 1,
        })
        if #entities == 0 then c.reply("No resource found at position") return end
        entities[1].amount = math.floor(amount)
        c.reply(string.format("Set %s amount=%d", entities[1].name, entities[1].amount))
    end)

    commands.add_command("pollution-get", "Get pollution at position. Usage: /pollution-get x,y [surface]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 1 then c.reply("Usage: /pollution-get x,y [surface]") return end
        local pos = c.parse_position(args[1])
        if not pos then c.reply("Invalid position. Use: x,y") return end
        local surface = c.get_surface(args[2])
        if not surface then c.reply("Surface not found") return end
        c.reply(string.format("%.2f", surface.get_pollution(pos)))
    end)

    commands.add_command("pollution-set", "Set pollution at position. Usage: /pollution-set x,y amount [surface]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 2 then c.reply("Usage: /pollution-set x,y amount [surface]") return end
        local pos = c.parse_position(args[1])
        if not pos then c.reply("Invalid position. Use: x,y") return end
        local amount = tonumber(args[2])
        if not amount then c.reply("Invalid amount") return end
        local surface = c.get_surface(args[3])
        if not surface then c.reply("Surface not found") return end
        surface.set_pollution(pos, amount)
        c.reply(string.format("Set pollution=%.2f", amount))
    end)

    commands.add_command("pollution-total", "Get total pollution. Usage: /pollution-total [surface]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        local surface = c.get_surface(args[1])
        if not surface then c.reply("Surface not found") return end
        c.reply(string.format("%.2f", surface.get_total_pollution()))
    end)

end

resource_mod.on_init = register_commands
resource_mod.on_load = register_commands

return resource_mod
