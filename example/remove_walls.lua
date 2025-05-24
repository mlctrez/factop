local surface = game.surfaces["nauvis"]

-- deletes all player walls
local d = 800
local area = { { -d, -d }, { d, d } }
area = nil
for _, entity in pairs(surface.find_entities_filtered({
    area = area,
    name = { "stone-wall","gate" },
})) do
    entity.destroy()
end
