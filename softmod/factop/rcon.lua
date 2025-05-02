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
    }
    rcon.print(helpers.table_to_json(result))
end
