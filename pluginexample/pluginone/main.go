// Plugin pluginone subscribes to player movement events and places random
// lab tiles under each player's position as they walk around.
package main

import (
	"fmt"
	"math"
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/mlctrez/factop/client/tile"
	"github.com/mlctrez/factop/plugin"
)

// labTiles are the lab tile variants available in Factorio.
var labTiles = []string{"lab-dark-1", "lab-dark-2", "lab-white"}

type labTrail struct{}

func (l *labTrail) Setup(ctx plugin.Context) error {
	ctx.Logger().Info("setup complete")
	return nil
}

func (l *labTrail) Run(ctx plugin.Context) error {
	log := ctx.Logger()
	tileClient := ctx.Clients().Tile()

	err := ctx.Subscribe("udp.incoming", func(data []byte) {
		msg := string(data)

		// Movement messages have the format:
		//   [move] #<index> <name> to {x=<x>, y=<y>}
		if !strings.HasPrefix(msg, "[move]") {
			return
		}

		x, y, err := parseMove(msg)
		if err != nil {
			log.Error("failed to parse move", "msg", msg, "error", err)
			return
		}

		tx := int(math.Floor(x))
		ty := int(math.Floor(y))

		name := labTiles[rand.IntN(len(labTiles))]
		// Area uses exclusive right_bottom, so +1 to cover the single tile.
		area := tile.Area{X1: tx, Y1: ty, X2: tx + 1, Y2: ty + 1}
		if _, err := tileClient.Fill(area, name, ""); err != nil {
			log.Error("failed to place tile", "error", err)
		}
	})
	if err != nil {
		return fmt.Errorf("subscribe udp.incoming: %w", err)
	}

	log.Info("running")
	<-ctx.Done()
	log.Info("shutting down")
	return nil
}

// parseMove extracts x,y from a movement message like:
//
//	[move] #1 PlayerName to {x=12.3, y=-4.5}
func parseMove(msg string) (float64, float64, error) {
	idx := strings.Index(msg, "{x=")
	if idx < 0 {
		return 0, 0, fmt.Errorf("no position block found")
	}
	inner := msg[idx+1 : len(msg)-1] // strip { and }
	parts := strings.Split(inner, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected 2 components, got %d", len(parts))
	}
	xStr := strings.TrimPrefix(strings.TrimSpace(parts[0]), "x=")
	x, err := strconv.ParseFloat(xStr, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parse x: %w", err)
	}
	yStr := strings.TrimPrefix(strings.TrimSpace(parts[1]), "y=")
	y, err := strconv.ParseFloat(yStr, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parse y: %w", err)
	}
	return x, y, nil
}

func main() {
	plugin.Run(&labTrail{})
}
