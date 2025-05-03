local surface = game.get_surface(1)

local loco = surface.create_entity({
    force = "player", name = "locomotive", position = { 26, 0 },
    create_build_effect_smoke = false, direction = defines.direction.south,
    move_stuck_players = true
})
if loco ~= nil then
    loco.destructible = false
    loco.insert({ name = "nuclear-fuel", count = 5 })
    local character = surface.create_entity{ name = "character", position = loco.position, force = "player" }
    loco.set_driver(character)
    character.riding_state = { acceleration = defines.riding.acceleration.accelerating, direction = defines.riding.direction.straight }
    for _, t in pairs(game.train_manager.get_trains({force="player"})) do
        t.speed = 2
    end
end

