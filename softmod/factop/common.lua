-- Common helper functions shared across factop softmod modules.
-- This file is required directly by other modules and should NOT be
-- registered with the event handler (it has no events).

local common = {}

--- Get a surface by name, defaulting to nauvis.
function common.get_surface(name)
    if name and name ~= "" then
        return game.surfaces[name]
    end
    return game.surfaces["nauvis"]
end

--- Parse "x1,y1,x2,y2" into left_top and right_bottom tables (integer coords).
-- Ensures left_top < right_bottom by sorting.
function common.parse_tile_area(s)
    local x1, y1, x2, y2 = s:match("^([%-]?%d+),([%-]?%d+),([%-]?%d+),([%-]?%d+)$")
    if not x1 then return nil end
    x1, y1, x2, y2 = tonumber(x1), tonumber(y1), tonumber(x2), tonumber(y2)
    return { x = math.min(x1, x2), y = math.min(y1, y2) },
           { x = math.max(x1, x2), y = math.max(y1, y2) }
end

--- Parse "x1,y1,x2,y2" into a BoundingBox table (float coords).
function common.parse_area(s)
    local x1, y1, x2, y2 = s:match("^([%-]?%d+%.?%d*),([%-]?%d+%.?%d*),([%-]?%d+%.?%d*),([%-]?%d+%.?%d*)$")
    if not x1 then return nil end
    x1, y1, x2, y2 = tonumber(x1), tonumber(y1), tonumber(x2), tonumber(y2)
    return { { math.min(x1, x2), math.min(y1, y2) }, { math.max(x1, x2), math.max(y1, y2) } }
end

--- Parse "x,y" into a position table.
function common.parse_position(s)
    local px, py = s:match("^([%-]?%d+%.?%d*),([%-]?%d+%.?%d*)$")
    if not px then return nil end
    return { x = tonumber(px), y = tonumber(py) }
end

return common
