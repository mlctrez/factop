local amount = 6000

-- script that makes some pollution at the center of the map

game.surfaces[1].pollute({ -5, -5 }, amount)
game.surfaces[1].pollute({ 5, -5 }, amount)
game.surfaces[1].pollute({ 5, 5 }, amount)
game.surfaces[1].pollute({ -5, 5 }, amount)

