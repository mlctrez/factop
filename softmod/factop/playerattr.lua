local playerattr_mod = {}
local c = require("factop.common")

-- Generic per-player key-value attribute storage.
-- Stores attributes in storage.playerattr[player_index] as flat string→string maps.

local function setup()
    if storage.playerattr == nil then
        storage.playerattr = {}
    end
end

local function get_player_attrs(player_index)
    setup()
    if storage.playerattr[player_index] == nil then
        storage.playerattr[player_index] = {}
    end
    return storage.playerattr[player_index]
end

local function resolve_player(name_or_index)
    local player = game.get_player(name_or_index)
    if not player then
        local idx = tonumber(name_or_index)
        if idx then player = game.players[idx] end
    end
    return player
end

local function register_commands()

    commands.add_command("playerattr-set", "Set player attribute. Usage: /playerattr-set player key value", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 3 then c.reply("Usage: /playerattr-set player key value") return end
        local player = resolve_player(args[1])
        if not player then c.reply("Player not found: " .. args[1]) return end
        local attrs = get_player_attrs(player.index)
        attrs[args[2]] = table.concat(args, " ", 3)
        c.reply("ok")
    end)

    commands.add_command("playerattr-get", "Get player attribute. Usage: /playerattr-get player key", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 2 then c.reply("Usage: /playerattr-get player key") return end
        local player = resolve_player(args[1])
        if not player then c.reply("Player not found: " .. args[1]) return end
        local attrs = get_player_attrs(player.index)
        c.reply(attrs[args[2]] or "")
    end)

    commands.add_command("playerattr-del", "Delete player attribute. Usage: /playerattr-del player key", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 2 then c.reply("Usage: /playerattr-del player key") return end
        local player = resolve_player(args[1])
        if not player then c.reply("Player not found: " .. args[1]) return end
        local attrs = get_player_attrs(player.index)
        attrs[args[2]] = nil
        c.reply("ok")
    end)

    commands.add_command("playerattr-list", "List player attributes. Usage: /playerattr-list player", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 1 then c.reply("Usage: /playerattr-list player") return end
        local player = resolve_player(args[1])
        if not player then c.reply("Player not found: " .. args[1]) return end
        local attrs = get_player_attrs(player.index)
        local parts = {}
        for k, v in pairs(attrs) do
            parts[#parts + 1] = k .. ":" .. v
        end
        c.reply(table.concat(parts, ","))
    end)

    commands.add_command("playerattr-keys", "List player attribute keys. Usage: /playerattr-keys player", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 1 then c.reply("Usage: /playerattr-keys player") return end
        local player = resolve_player(args[1])
        if not player then c.reply("Player not found: " .. args[1]) return end
        local attrs = get_player_attrs(player.index)
        local keys = {}
        for k, _ in pairs(attrs) do
            keys[#keys + 1] = k
        end
        c.reply(table.concat(keys, ","))
    end)

end

local function on_player_joined(event)
    get_player_attrs(event.player_index)
end

playerattr_mod.events = {
    [defines.events.on_player_joined_game] = on_player_joined,
}

playerattr_mod.on_init = register_commands
playerattr_mod.on_load = register_commands

return playerattr_mod
