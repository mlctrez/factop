local surface = game.get_surface(1)
local items = surface.find_entities_filtered { to_be_deconstructed = true }
for _, item in pairs(items) do
    item.destroy()
end
