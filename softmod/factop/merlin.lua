wizard_merlin = {}

wizard_merlin.summon = function(player)
    local merlin = nil
    -- todo: add support for a merlin per player with name_tag = merlin_<player.id> ?
    for _, c in pairs(player.surface.find_entities_filtered { name = "character", player.force }) do
        if c.name_tag == "merlin" then
            merlin = c
        end
    end

    -- find a place for merlin to appear that does not collide with player or surrounding entities
    local appear_at = player.surface.find_non_colliding_position("character", player.position, 10, 1, false)
    if appear_at == nil then
        player.print({ message = "merlin.summon.failed" })
        return
    end

    local console_message = { message = "merlin.summon.existing" }
    if merlin == nil then
        console_message = { message = "merlin.summon.created" }
        merlin = player.surface.create_entity {
            name = "character", force = player.force,
            position = appear_at, move_stuck_players = true
        }
        merlin.name_tag = "merlin"
        merlin.color = { 0, 0, 0, 255 } -- none more black
        merlin.destructible = false -- enemies and players cannot destroy
        merlin.operable = false -- prevent stealing inventory or armor
        -- todo: add inventory, armor, weapons
    else
        merlin.teleport(appear_at, player.surface, false, false)
    end
    -- merlin appears with a bang
    player.surface.create_entity { name = "atomic-fire-smoke", force = player.force, position = appear_at }

    player.print(console_message)
    return merlin
end

-- handles console commands for merlin
-- The structure of ccd defined at
-- https://lua-api.factorio.com/latest/concepts/CustomCommandData.html
wizard_merlin.command = function(ccd)
    local player = game.players[ccd.player_index]
    if ccd.parameter == "summon" then
        wizard_merlin.summon(player)
    elseif ccd.parameter == "follow" then
        player.print({ message = "merlin.not.implemented" })
    elseif ccd.parameter == "fight" then
        player.print({ message = "merlin.not.implemented" })
    else
        player.print({ message = "merlin.command.bad" })
    end
end

wizard_merlin.add_commands = function()
    commands.add_command("merlin", { message = "merlin.help" }, wizard_merlin.command)
end

return wizard_merlin