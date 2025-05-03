package main

import (
	"fmt"
	"github.com/mlctrez/factop/example"
	"math"
	"math/rand"
	"time"
)

func main() {
	ex := example.New()
	defer ex.Close()
	spawner := time.NewTicker(2 * time.Second)
	clearer := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-spawner.C:
			spider(ex)
		case <-clearer.C:
			clearGhosts(ex)
		}
	}
}

func clearGhosts(ex *example.Example) {
	cmd := `
local surface = game.surfaces["nauvis"]
local ghosts = surface.find_entities_filtered{type="entity-ghost"}
for _, ghost in pairs(ghosts) do
	if ghost.ghost_name == "spidertron" then
		ghost.destroy()
	end
end`
	ex.RconSc(cmd)
}

func spider(ex *example.Example) {
	r := rand.Float64() * 2 * math.Pi

	var startDistance float64 = -30
	var endDistance float64 = 100

	color := fmt.Sprintf("{ r = %1.0f, g = %1.0f, b = %1.0f, a = 1 }", rand.Float64(), rand.Float64(), rand.Float64())

	startPosition := fmt.Sprintf("{ %3.2f, %3.2f }", math.Cos(r)*startDistance, math.Sin(r)*startDistance)
	endPosition := fmt.Sprintf("{ %3.2f, %3.2f }", math.Cos(r)*endDistance, math.Sin(r)*endDistance)

	skeletons := rand.Intn(3) + 1

	cmd := fmt.Sprintf(`
if #game.connected_players > 0 then
	local surface = game.surfaces["nauvis"]
	local spider = surface.create_entity {
		name = "spidertron", position = %s, force = "player", 
		create_build_effect_smoke = false, move_stuck_players = true
	}
	spider.grid.put { name = "fusion-reactor-equipment", quality="legendary"}
	spider.grid.put({ name = "personal-laser-defense-equipment"})
	for i = 1, %d do
		spider.grid.put({ name = "exoskeleton-equipment", quality="legendary"})
	end
	spider.add_autopilot_destination %s
	spider.color = %s
end
`, startPosition, skeletons, endPosition, color)
	ex.RconSc(cmd)
}
