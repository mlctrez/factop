wizard_merlin = {}

local merlin_data = "merlin_data"

wizard_merlin.find_merlin = function(surface)
    for _, c in pairs(surface.find_entities_filtered { name = "character", force = "player" }) do
        if c.name_tag == "merlin" then
            return c
        end
    end
    return nil
end

wizard_merlin.summon = function(player)
    -- todo: add support for a merlin per player with name_tag = merlin_<player.id> ?
    local merlin = wizard_merlin.find_merlin(player.surface)

    -- clear existing follow data
    factop_storage.put(merlin_data, player.surface.name)

    -- find a place for merlin to appear that does not collide with player or surrounding entities
    local appear_at = player.surface.find_non_colliding_position("character", player.position, 10, 1, false)
    if appear_at == nil then
        player.print({ "merlin.summon.failed" })
        return
    end

    local console_message = { "merlin.summon.existing" }
    if merlin == nil then
        console_message = { "merlin.summon.created" }
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
    player.surface.create_entity {
        name = "promethium-asteroid-explosion-5",
        force = player.force, position = appear_at }

    player.print(console_message)
    return merlin
end

wizard_merlin.follow = function(player)
    factop_storage.put(merlin_data, player.surface.name, { following = player.index })
    player.print({ "merlin.message.following" })
end

wizard_merlin.goaway = function(player)
    local merlin = wizard_merlin.find_merlin(player.surface)
    if merlin ~= nil then
        player.print({ "merlin.message.goaway" })
        wizard_merlin.cleanup(merlin, player.surface.name)
        merlin.destroy()
    end
end
wizard_merlin.cleanup = function(merlin, surface_name)
    merlin.walking_state = { walking = false }
    factop_storage.put(merlin_data, surface_name)
end

wizard_merlin.periodic = function()
    -- need to track who merlin is following on each given surface
    local data = storage[merlin_data]
    if data == nil then
        -- to early
        return
    end
    for _, surface_name in pairs(factop_storage.keys(data)) do
        local s = game.surfaces[surface_name]
        if s == nil then
            return wizard_merlin.cleanup(merlin, surface_name)
        end
        local follow_data = factop_storage.get(merlin_data, surface_name)
        if follow_data ~= nil then
            local merlin = wizard_merlin.find_merlin(s)
            if merlin == nil then
                return
            end
            -- move merlin towards player location
            local player_index = follow_data.following
            local player = game.players[player_index]
            if player == nil or not player.valid or not player.connected then
                return wizard_merlin.cleanup(merlin, surface_name)
            end

            local distance = factop_flib.position.distance(merlin.position, player.position)
            if distance > 60 then
                player.print({ "merlin.message.distance" })
                return wizard_merlin.cleanup(merlin, surface_name)
            end

            if distance < 5 then
                merlin.walking_state = { walking = false }
            else
                local walking_direction = factop_flib.direction.from_positions(merlin.position, player.position)
                merlin.walking_state = { walking = true, direction = walking_direction }
            end
        end
    end
end

-- handles console commands for merlin
-- The structure of ccd defined at
-- https://lua-api.factorio.com/latest/concepts/CustomCommandData.html
wizard_merlin.command = function(ccd)
    local player = game.players[ccd.player_index]
    if ccd.parameter == "summon" then
        wizard_merlin.summon(player)
    elseif ccd.parameter == "goaway" then
        wizard_merlin.goaway(player)
    elseif ccd.parameter == "follow" then
        wizard_merlin.follow(player)
    elseif ccd.parameter == "fight" then
        player.print({ "merlin.not.implemented" })
    else
        player.print({ "merlin.command.bad" })
    end
end

wizard_merlin.add_commands = function()
    commands.add_command("merlin", { "merlin.help" }, wizard_merlin.command)
end

wizard_merlin.on_nth_tick = {
    [20] = wizard_merlin.periodic
}

return wizard_merlin