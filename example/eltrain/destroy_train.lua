local surface = game.get_surface(1)

for _, ent in pairs(surface.find_entities_filtered({
    name = { "locomotive" },
    area = { { -150, -150 }, { 150, 150 } },
    force = "player"
})) do
    if ent ~= nil then
        ent.destroy()
    end
end

