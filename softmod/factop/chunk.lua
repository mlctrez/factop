factop_chunk = {}

-- https://lua-api.factorio.com/latest/events.html#on_chunk_generated
factop_chunk.generated = function(event)
    print("factop_chunk.generated ".. serpent.line(event.position))
end

-- https://lua-api.factorio.com/latest/events.html#on_chunk_charted
factop_chunk.charted = function(event)
    --print("factop_chunk.charted ".. serpent.line(event.position))
end


factop_chunk.generated_older = function(event)
    local position = event.position
    -- skip for chunks inside this radius
    local chunk_radius = 2;
    local cx = position.x
    local cy = position.y
    if cx < 0 then
        cx = cx + 1
    end
    if cy < 0 then
        cy = cy + 1
    end
    if math.abs(cx) < chunk_radius and math.abs(cy) < chunk_radius then
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


factop_chunk.events = {
    [defines.events.on_chunk_generated] = factop_chunk.generated,
    [defines.events.on_chunk_charted] = factop_chunk.charted
}

-- commented out for now, as it interferes with free play chunk generation on nauvis
return factop_chunk