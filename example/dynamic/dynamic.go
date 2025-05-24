package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/mlctrez/factop/api"
	"github.com/nats-io/nats.go"
	"log"
	"math"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

//go:embed dynamic.lua
var script string

type Dynamic struct {
	client          *api.RconClient
	playerPositions map[string]Position
}

func (e *Dynamic) execJson(extra string, bindTo any) error {
	payload := fmt.Sprintf("/sc %s\n\n%s", script, extra)
	rconResult, err := e.client.Execute(&api.RconCommand{Payload: payload})
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(rconResult.Payload), bindTo)
	if err != nil {
		fmt.Println(rconResult.Payload)
	}
	return err
}

func (e *Dynamic) Run() {
	findTilesData := &FindTiles{}
	err := e.execJson(`findTiles("nauvis", {64,32}, 4)`, findTilesData)
	if err != nil {
		log.Println(err)
		return
	}
	if findTilesData.Error != "" {
		log.Println(findTilesData.Error)
		return
	}
	fmt.Println(findTilesData)
}

func (e *Dynamic) LabTiles() {
	payload := fmt.Sprintf("/sc %s\n\n%s", script, "labTiles()")
	_, err := e.client.Execute(&api.RconCommand{Payload: payload})
	if err != nil {
		log.Println(err)
	}
}

func (e *Dynamic) Players() {
	payload := fmt.Sprintf("/sc factop_rcon.players()")
	response, err := e.client.Execute(&api.RconCommand{Payload: payload})
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(response.Payload)
}

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Player struct {
	Name     string   `json:"name"`
	Position Position `json:"position"`
}

func (e *Dynamic) playerMoved() {
	payload := fmt.Sprintf("/sc %s\n\n%s", script, "playerPositions()")
	response, err := e.client.Execute(&api.RconCommand{Payload: payload})
	if err != nil {
		log.Println(err)
		return
	}

	var players []Player
	if err := json.Unmarshal([]byte(response.Payload), &players); err != nil {
		log.Printf("Error unmarshaling player positions: %v", err)
		return
	}

	// Update positions for all players and check for significant movement
	for _, player := range players {
		if lastPos, exists := e.playerPositions[player.Name]; exists {
			// Check if movement is more than 2 units on either axis
			deltaX := math.Abs(player.Position.X - lastPos.X)
			deltaY := math.Abs(player.Position.Y - lastPos.Y)

			if deltaX > 2 || deltaY > 2 {
				fmt.Printf("player %s moved from (%.2f, %.2f) to (%.2f, %.2f)\n",
					player.Name,
					lastPos.X, lastPos.Y,
					player.Position.X, player.Position.Y)
			}
		}

		// Update the position in the map
		e.playerPositions[player.Name] = player.Position
	}
}

func main() {
	d := &Dynamic{}
	d.playerPositions = make(map[string]Position)
	log.Default().SetFlags(0)
	if con, err := nats.Connect("nats://factorio"); err != nil {
		log.Fatal(err)
	} else {
		sub, errSub := con.Subscribe("factorio.stdout", func(m *nats.Msg) {
			l := strings.TrimSpace(string(m.Data))
			if strings.Contains(l, "factop_player.position_updated") {
				//d.playerMoved()
			}
		})
		if errSub != nil {
			log.Fatal(errSub)
		}
		d.client = api.NewRconClient(con)
		d.Players()
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		_ = sub.Unsubscribe()
		con.Close()
	}
}

type FindTiles struct {
	Count int    `json:"count"`
	Tiles []Tile `json:"tiles,omitempty"`
	Error string `json:"error,omitempty"`
}

type Tile struct {
	Name     string   `json:"name"`
	Position Position `json:"position"`
}

func (f *FindTiles) String() string {
	result := fmt.Sprintf("Total tiles: %d\n", f.Count)
	limit := 10
	if len(f.Tiles) < limit {
		limit = len(f.Tiles)
	}
	for i := 0; i < limit; i++ {
		tile := f.Tiles[i]
		result += fmt.Sprintf("Tile %d: %s at position (%d,%d)\n", i+1, tile.Name, tile.Position.X, tile.Position.Y)
	}
	return result
}
