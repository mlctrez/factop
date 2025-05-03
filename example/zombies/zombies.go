package main

import (
	"fmt"
	"github.com/mlctrez/factop/example"
	"math"
	"math/rand/v2"
	"time"
)

func main() {
	ex := example.New()
	ex.DontExitOnError = true
	defer ex.Close()
	deleteCorpses := time.Now().Add(30 * time.Second)
	for {
		distance := rand.Float64()*10 + 18
		angle := rand.Float64() * 2 * math.Pi
		pos := fmt.Sprintf("{%2.2f, %2.2f}", math.Sin(angle)*distance, math.Cos(angle)*distance)
		color := fmt.Sprintf("{ %1.2f, %1.2f,  %1.2f, 1}", rand.Float64(), rand.Float64(), rand.Float64())
		force := "enemy"
		payload := fmt.Sprintf(`if factop_player.total_connected() > 0 then
	local c = game.surfaces[1].create_entity { 
		name = "character", position = %s, force = %q, 
	move_stuck_players = true}
	if  c ~= nil then
		c.health = 150
		c.color = %s
		local  ch_armor = c.get_inventory(defines.inventory.character_armor)
		local stack = ch_armor[1]
		stack.set_stack { name = "power-armor-mk2" }
		stack.grid.put({ name = "fusion-reactor-equipment"})
		stack.grid.put({ name = "personal-laser-defense-equipment"})
	end
end
`, pos, force, color)
		ex.RconSc(payload)
		//fmt.Println(payload)
		time.Sleep(2 * time.Second)

		if time.Now().After(deleteCorpses) {
			deleteCorpses = time.Now().Add(30 * time.Second)
			// clean map area of corpses or characters not attached to a player
			ex.RconSc(`local surface = game.surfaces["nauvis"]

local d = 200
for _, entity in pairs(surface.find_entities_filtered({
    area = { { -d, -d }, { d, d } },
    name = { "character-corpse" },
})) do
    entity.destroy()
end

for _, entity in pairs(surface.find_entities_filtered({
    area = { { -d, -d }, { d, d } },
    name = { "character" },
})) do
    local playerChar = false
    for _, player in pairs(game.players) do
        if player.character == entity then
            playerChar = true
        end
    end
    if not playerChar then
        entity.destroy()
    end
end

`)
		}
	}

}
