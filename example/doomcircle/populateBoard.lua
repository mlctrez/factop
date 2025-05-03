local surface = game.surfaces["nauvis"]

--factop_tiles.create(-192, 191, -192, 191, "nauvis", "refined-concrete")

surface.create_entity({
    name = "radar", position = { 150, 0 }, force = "player", quality = "legendary",
    create_build_effect_smoke = false
})
surface.create_entity({
    name = "radar", position = { -150, 0 }, force = "player", quality = "legendary",
    create_build_effect_smoke = false
})

local eei = surface.create_entity({
    name = "electric-energy-interface", position = { 0, 150 }, force = "enemy",
    create_build_effect_smoke = false, snap_to_grid = true, move_stuck_players = true
})
if eei ~= nil then
    eei.destructible = false
    eei.minable = false
    eei.operable = false
    eei.power_production = 7000000000
    eei.power_usage = 0
    eei.electric_buffer_size = 10000000000
end
