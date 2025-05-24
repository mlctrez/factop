local surface = game.surfaces[1]

local car = surface.create_entity({
    force = "player", name = "car",
    quality = "legendary",
    position = { 16, 16 },
    create_build_effect_smoke = false,
    direction = defines.direction.southwest,
    move_stuck_players = true
})
if car ~= nil then
    car.insert({ name = "nuclear-fuel", count = 5 })
    car.insert({ name = "repair-pack", count = 100 })
    --local character = surface.create_entity{ name = "character", position = car.position, force = "player" }
    --car.set_driver(character)
    --character.riding_state = { acceleration = defines.riding.acceleration.accelerating, direction = defines.riding.direction.right }
    --car.speed = 2
end

