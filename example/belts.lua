local s = game.surfaces["nauvis"]
local name = "turbo-transport-belt"
local force = "player"

for _, ent in pairs(s.find_entities_filtered {
    name = name, force = force
}) do
    ent.destroy()
end

local ty = 14.5

for dx = -12, 10, 2 do
    s.create_entity { name = name, force = force, position = { 16.5 + dx, ty + 1 }, direction = defines.direction.north }
    s.create_entity { name = name, force = force, position = { 16.5 + dx, ty }, direction = defines.direction.east }
    s.create_entity { name = name, force = force, position = { 16.5 + dx + 1, ty }, direction = defines.direction.south }
    local last = s.create_entity { name = name, force = force, position = { 16.5 + dx + 1, ty + 1 }, direction = defines.direction.east }
    if dx == 10 then
        last.rotate()
    end
end

for dx = 10, -12, -2 do
    s.create_entity { name = name, force = force, position = { 17.5 + dx, ty + 2 }, direction = defines.direction.south }
    s.create_entity { name = name, force = force, position = { 17.5 + dx, ty + 3 }, direction = defines.direction.west }
    s.create_entity { name = name, force = force, position = { 17.5 + dx - 1, ty + 3 }, direction = defines.direction.north }
    local last = s.create_entity { name = name, force = force, position = { 17.5 + dx - 1, ty + 2 }, direction = defines.direction.west }
    if dx == -12 then
        last.rotate()
    end
end

s.spill_item_stack { position = { 16, 16 }, stack = { name = "iron-plate", count = 2000, quality="legendary" } }
local items = s.find_entities_filtered { name = "item-on-ground", position = { 0, 0 }, radius = 100 }
for _, item in pairs(items) do
    item.destroy()
end

