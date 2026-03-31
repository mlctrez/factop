local game_mod = {}
local c = require("factop.common")

-- Game-level server administration module for factop softmod.
-- Wraps LuaGameScript operations that are not tied to a specific
-- surface, entity, or player lifecycle.

-- ---------------------------------------------------------------------------
-- Console commands
-- ---------------------------------------------------------------------------

local function register_commands()

    -- /game-players
    commands.add_command("game-players", "List connected players. Usage: /game-players", function(cmd)
        if not c.rcon_only(cmd) then return end
        local parts = {}
        for _, player in pairs(game.connected_players) do
            local x, y = 0, 0
            local has_char = "false"
            if player.character and player.character.valid then
                x = player.character.position.x
                y = player.character.position.y
                has_char = "true"
            elseif player.physical_position then
                x = player.physical_position.x
                y = player.physical_position.y
            end
            parts[#parts + 1] = string.format("%s:%d:%.1f:%.1f:%s",
                player.name, player.index, x, y, has_char)
        end
        c.reply(table.concat(parts, ","))
    end)

    -- /game-players-all
    -- Returns all players (connected and disconnected).
    -- Wire format: name:index:connected,...
    commands.add_command("game-players-all", "List all players. Usage: /game-players-all", function(cmd)
        if not c.rcon_only(cmd) then return end
        local parts = {}
        for _, player in pairs(game.players) do
            parts[#parts + 1] = string.format("%s:%d:%s",
                player.name, player.index, tostring(player.connected))
        end
        c.reply(table.concat(parts, ","))
    end)

    -- /game-kick <player> [reason]
    commands.add_command("game-kick", "Kick a player. Usage: /game-kick player [reason]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 1 then c.reply("Usage: /game-kick player [reason]") return end

        local player = game.get_player(args[1])
        if not player then
            local idx = tonumber(args[1])
            if idx then player = game.players[idx] end
        end
        if not player then c.reply("Player not found: " .. args[1]) return end

        local reason = nil
        if #args >= 2 then
            reason = table.concat(args, " ", 2)
        end

        game.kick_player(player, reason)
        c.reply("Kicked " .. player.name)
    end)

    -- /game-remove <player>
    -- Kicks the player if connected, then removes them from the save.
    -- On reconnect they will start as a new player.
    commands.add_command("game-remove", "Remove player from save. Usage: /game-remove player", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 1 then c.reply("Usage: /game-remove player") return end

        local player = game.get_player(args[1])
        if not player then
            local idx = tonumber(args[1])
            if idx then player = game.players[idx] end
        end
        if not player then c.reply("Player not found: " .. args[1]) return end

        local name = player.name

        -- Kick first if connected
        if player.connected then
            game.kick_player(player, "Removed from save")
        end

        -- Remove from save (only works on offline players)
        game.remove_offline_players({ player })
        c.reply("Removed " .. name)
    end)

end

game_mod.on_init = register_commands
game_mod.on_load = register_commands

return game_mod
