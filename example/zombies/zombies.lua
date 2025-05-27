zombies = {}

zombies.enemy = function()
    return { force = "enemy", color = { 1, 0, 0, 1 } }
end
zombies.player = function()
    return { force = "player", color = { 0, 1, 0, 1 } }
end

zombies.move = function()
    if factop_player.total_connected() == 0 then
        return
    end

    local surface = game.surfaces["nauvis"]
    for _, entity in pairs(surface.find_entities_filtered({
        position = { 0, 0 }, radius = 200, name = { "character" } })) do
        local playerChar = false
        for _, player in pairs(game.players) do
            if player.character == entity then
                playerChar = true
            end
        end
        if not playerChar then
            entity.walking_state = { walking = true, direction = math.random() * 16 }
        end
    end

end

zombies.create = function()
    if factop_player.total_connected() == 0 then
        return
    end

        local surface = game.surfaces["nauvis"]

        local d = math.random() * 10 + 25
        local a = math.random() * 2 * math.pi
        local pos = { 16 + math.sin(a) * d, 16 + math.cos(a) * d }
        local data = zombies.player()
        if math.random() > 0.5 then
            data = zombies.enemy()
        end
        local z = surface.create_entity {
            name = "character", position = pos, force = data.force, move_stuck_players = true }
        if z ~= nil then
            z.health = 100
            z.color = data.color
            local ch_armor = z.get_inventory(defines.inventory.character_armor)
            local stack = ch_armor[1]
            stack.set_stack { name = "power-armor-mk2" }
            stack.grid.put({ name = "fusion-reactor-equipment" })
            stack.grid.put({ name = "personal-laser-defense-equipment" })
            stack.grid.put({ name = "personal-laser-defense-equipment" })
            stack.grid.put({ name = "personal-laser-defense-equipment" })
        end
        for _, entity in pairs(surface.find_entities_filtered({
            position = { 0, 0 }, radius = 200, name = { "character-corpse" } })) do
            entity.destroy()
        end
end

zombies.delete = function()

    local surface = game.surfaces["nauvis"]
    for _, entity in pairs(surface.find_entities_filtered({
        position = { 0, 0 }, radius = 200,
        name = { "character", "character-corpse" },
    })) do
        local playerChar = false
        for _, player in pairs(game.players) do
            if player.character == entity then
                playerChar = true
            end
        end
        if not playerChar then
            entity.destroy()
        end
    end
end