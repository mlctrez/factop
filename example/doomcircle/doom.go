package main

import (
	_ "embed"
	"fmt"
	"github.com/mlctrez/factop/example"
	"math"
	"time"
)

//go:embed resetBoard.lua
var resetBoard string

//go:embed populateBoard.lua
var populateBoard string

func main() {

	ex := example.New()
	defer ex.Close()

	ex.RconSc(resetBoard)
	time.Sleep(20 * time.Second)
	ex.RconSc(populateBoard)

	var r float64
	var distance float64 = 90
	for r = 0; r <= 2*math.Pi; r += (2 * math.Pi) / 100 {
		command := `game.surfaces["nauvis"].create_entity({
		name="tesla-turret",
		position={%3.1f,%3.1f},
		quality="legendary",
		force="enemy",
		create_build_effect_smoke=false,
		snap_to_grid=false,
		move_stuck_players=true
	 })`
		out := fmt.Sprintf(command, math.Cos(r)*distance, math.Sin(r)*distance)
		ex.RconSc(out)
		time.Sleep(10 * time.Millisecond)
	}

	distance = 41
	spawnerType := "biter-spawner"
	for r = 0; r <= 2*math.Pi; r += (2 * math.Pi) / 30 {
		command := `
	local spawner = game.surfaces["nauvis"].create_entity({
		name=%q,
		position={%3.1f,%3.1f},
		quality="legendary",
		force="player",
		create_build_effect_smoke=false,
		snap_to_grid=false,
		move_stuck_players=true
	 })
	 if spawner ~= nil then
	 	spawner.destructible = false
	 end`
		out := fmt.Sprintf(command, spawnerType, math.Cos(r)*distance, math.Sin(r)*distance)
		ex.RconSc(out)
		time.Sleep(200 * time.Millisecond)

		// alternate spawner types
		if spawnerType == "biter-spawner" {
			spawnerType = "spitter-spawner"
		} else {
			spawnerType = "biter-spawner"
		}

	}

	// the external wall around the tesla turrets was nice but took too long to generate
	//distance = 104
	//for r = 0; r <= 2*math.Pi; r += (2 * math.Pi) / 1500 {
	//	command := `factop_wall.create_barrier("nauvis",%3.1f,%3.1f)`
	//	ex.RconSc(fmt.Sprintf(command, math.Cos(r)*distance, math.Sin(r)*distance))
	//	ex.RconSc(fmt.Sprintf(command, math.Cos(r)*(distance+1), math.Sin(r)*(distance+1)))
	//	ex.RconSc(fmt.Sprintf(command, math.Cos(r)*(distance+2), math.Sin(r)*(distance+2)))
	//	time.Sleep(10 * time.Millisecond)
	//}
	//
	//time.Sleep(3 * time.Second)
}
