local surface = game.surfaces["nauvis"]
surface.create_global_electric_network()

surface.daytime = 0.5
surface.freeze_daytime = true

local working_area = { { -250, -250 }, { 250, 250 } }

for _, corpse in ipairs(surface.find_entities_filtered({ area = working_area, name = "character-corpse" })) do
    corpse.destroy()
end

for _, entity in pairs(surface.find_entities_filtered({ area = working_area, force = "enemy" })) do
    entity.destroy()
end

local clear_types = {
    "radar", "electric-energy-interface", "tesla-turret", "stone-wall",
    "biter-spawner", "small-biter", "medium-biter", "big-biter", "behemoth-biter",
    "spitter-spawner", "small-spitter", "medium-spitter", "big-spitter", "behemoth-spitter",
}

for _, ent in pairs(surface.find_entities_filtered { area = working_area, name = clear_types }) do
    ent.destroy()
end


