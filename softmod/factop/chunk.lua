factop_chunk = {}

factop_chunk.generated = function(event)
    local position = event.position
    -- only override chunk generation for tiles further from this radius
    local chunk_radius = 6;
    if position.x <= -chunk_radius or position.x >= (chunk_radius - 1) or
            position.y <= -chunk_radius or position.y >= (chunk_radius - 1) then
        return
    end
    -- generated chunks are filled with lab tiles due to surface settings
    -- so we replace them with out-of-map tiles to prevent player navigation
    local area = event.area
    local tiles = {}
    for x = area.left_top.x, area.right_bottom.x - 1 do
        for y = area.left_top.y, area.right_bottom.y - 1 do
            table.insert(tiles, { name = "out-of-map", position = { x + 0.5, y + 0.5 } })
        end
    end
    event.surface.set_tiles(tiles, true)
end

--commented out for now, as it interferes with free play chunk generation on nauvis
--factop_chunk.events = {
--    [defines.events.on_chunk_generated] = factop_chunk.generated
--}

return factop_chunk