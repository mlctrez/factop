local createAsteroids = function(player, amount)
    if player == nil or not player.valid or not player.connected then
        return
    end

    local range = 4
    for i = 1, amount do
        local rad = (i / amount + (game.tick % 360) / 360) * 2 * math.pi
        local x_offset = range * math.cos(rad)
        local y_offset = range * math.sin(rad)
        player.surface.create_entity {
            name = "metallic-asteroid-explosion-3",
            position = { player.position.x + x_offset, player.position.y + y_offset },
            force = "player",
        }
    end
end

createAsteroids(game.players[1], 12)


