
-- example for creating a roboport with construction bots and repair packs
for _, ent in pairs(game.surfaces[1].find_entities_filtered({
    area = { { -10, -10 }, { 10, 10 } }, name = "roboport", force = "player"
})) do
    ent.destroy()
end

local port = game.surfaces[1].create_entity({
    name = "roboport", position = { 0, 0 }, force = game.forces.player,
    quality = "legendary", create_build_effect_smoke = false, move_stuck_players = true,
})
port.insert({ name = "construction-robot", count = 200, quality = "legendary" })
port.insert({ name = "repair-pack", count = 200, quality = "legendary" })

