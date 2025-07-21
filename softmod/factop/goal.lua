factop_goal = {}

factop_goal_setup = function()
    if storage.factop_goal == nil then
        storage.factop_goal = {}
    end
end

factop_goal.created = function(event)
    factop_goal_setup()
    if event.player_index then
        -- goal shown 1 second after player created
        table.insert(storage.factop_goal, {
            player_index = event.player_index, when = game.tick + 60, message = { "factop.restart.warning" }
        })
        -- goal hidden 20 seconds later
        table.insert(storage.factop_goal, {
            player_index = event.player_index, when = game.tick + 60 * 20, message = ""
        })
    end
end

factop_goal.periodic = function()
    factop_goal_setup()
    for i, data in pairs(storage.factop_goal) do
        if game.tick >= data.when then
            local player = game.players[data.player_index]
            if player ~= nil and player.valid and player.connected then
                -- show the goal dialog with the message, empty will close the dialog
                player.set_goal_description(data.message)
            end
            table.remove(storage.factop_goal, i)
        end
    end
end

factop_goal.events = {
    [defines.events.on_player_created] = factop_goal.created
}

factop_goal.on_nth_tick = {
    [60] = factop_goal.periodic
}

return factop_goal