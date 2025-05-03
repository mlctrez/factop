local s = game.surfaces["nauvis"]

-- make some asteroid explosions on nauvis
s.create_entity{name = "promethium-asteroid-explosion-5", position = {-10, 0}, force = "player"}
s.create_entity{name = "promethium-asteroid-explosion-5", position = {10, 0}, force = "player"}
