local surface = game.surfaces["nauvis"]

-- deletes all the NPC characters and corpses

local d = 200
local area = { { -d, -d }, { d, d } }
for _, entity in pairs(surface.find_entities_filtered({
    area = area,
    name = { "character-corpse" },
})) do
    entity.destroy()
end

for _, entity in pairs(surface.find_entities_filtered({
    area = area,
    name = { "character" },
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

