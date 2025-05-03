
-- example to force charting of chunks past a certain radius
local surface = game.surfaces["nauvis"];

game.players[1].teleport({ 0, 0 })
game.players[1].force.clear_chart()

game.forces["player"].cancel_charting(surface);
local chunk_radius = 5;
for chunk in surface.get_chunks() do
    if (chunk.x <= -chunk_radius or chunk.x >= chunk_radius or chunk.y <= -chunk_radius or chunk.y >= chunk_radius) then
        surface.delete_chunk(chunk)
    end
end

