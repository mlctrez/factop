local s = game.surfaces["nauvis"]

s.show_clouds = false
s.always_day = true

function fillArea(surface_name, left_top_x, left_top_y, right_bottom_x, right_bottom_y, tile_name)
    local surface = game.surfaces[surface_name]
    if not surface then
        return false
    end

    local tiles = {}
    for x = left_top_x, right_bottom_x do
        for y = left_top_y, right_bottom_y do
            table.insert(tiles, { name = tile_name, position = { x = x, y = y } })
        end
    end

    surface.set_tiles(tiles)
    return true
end

--local min = -32
--local max = 31
--fillArea("nauvis", min, min, max, max, "volcanic-ash-flats")

local items = s.find_entities_filtered { name = "item-on-ground", position = { 0, 0 }, radius = 100 }
for _, item in pairs(items) do
    item.destroy()
end

s.spill_item_stack { position = { 0, 0 }, stack = { name = "iron-plate", count = 200 } }
s.spill_item_stack { position = { 0, 0 }, stack = { name = "copper-plate", count = 400 } }
s.spill_item_stack { position = { 0, 0 }, stack = { name = "uranium-ore", count = 800 } }

error("leaving")

items = s.find_entities_filtered { name = "item-on-ground", position = { 0, 0 }, radius = 10 }

local found = {}
for _, item in pairs(items) do
    table.insert(found, { name = item.stack.name, gps = item.gps_tag })
end
rcon.print(helpers.table_to_json({ count = #items, found = found }))