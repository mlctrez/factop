-- the global holding all of the rcon functions
factop_rcon = {}

factop_rcon.ping = function(input)
    rcon.print(helpers.table_to_json({ input = input }))
end

factop_rcon.players = function()
    local players = {}
    for _, p in pairs(game.players) do
        table.insert(players, {
            index = p.index,
            connected = p.connected,
            name = p.name,
            admin = p.admin,
            position = p.position,
            afk_time = p.afk_time,
            online_time = p.online_time,
            last_online = p.last_online,
            force = p.force.name,
            color = p.color
        })
    end
    rcon.print(helpers.table_to_json({ players = players }))
end

factop_rcon.remove_offline = function(players)
    if type(players) ~= "table" then
        error("players is required and must be a table")
    end
    for k, v in pairs(players) do
        if type(k) ~= "number" or type(v) ~= "string" then
            error("table must be array of string")
        end
    end
    local removed = {}
    for _, name in pairs(players) do
        for _, p in pairs(game.players) do
            if p.name == name then
                game.remove_offline_players({ p })
                table.insert(removed, { name = name })
            end
        end
        -- force this one just for testing to get the error
        if name == "test_player_name_to_remove_bad" then
            game.remove_offline_players({ name })
        end
        if name == "test_player_name_to_remove_good" then
            table.insert(removed, { name = name })
        end
    end
    if #removed == 0 then
        -- helpers.table_to_json returns players = {} which is an object not
        -- an empty array, so just return an empty object to not break json parsing
        rcon.print("{}")
        return
    end
    rcon.print(helpers.table_to_json({ players = removed }))
end

factop_rcon.game = function(changes)
    if changes ~= nil then
        for k, v in pairs(changes) do
            if k == "tick_paused" then
                game.tick_paused = v
            elseif k == "autosave_enabled" then
                game.autosave_enabled = v
            elseif k == "speed" then
                game.speed = v
            elseif k == "ticks_to_run" then
                game.ticks_to_run = v
            end
        end
    end
    local result = {
        tick = game.tick,
        ticks_played = game.ticks_played,
        tick_paused = game.tick_paused,
        ticks_to_run = game.ticks_to_run,
        speed = game.speed,
        autosave_enabled = game.autosave_enabled,
        blueprints_count = #game.blueprints,
        connected_players = #game.connected_players
    }
    rcon.print(helpers.table_to_json(result))
end
