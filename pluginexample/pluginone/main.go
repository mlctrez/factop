// Plugin pluginone subscribes to player movement events and places random
// lab tiles under each player's position as they walk around.
package main

import (
	"fmt"
	"math"
	"math/rand/v2"
	"strings"

	"github.com/mlctrez/factop/client/player"
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
		//   [move] #<index> <name> <name>:<index>:<x>:<y>:<surface_name>:<surface_index>
		if !strings.HasPrefix(msg, "[move]") {
			return
		}

		mv, err := player.ParseMove(msg)
		if err != nil {
			log.Error("failed to parse move", "msg", msg, "error", err)
			return
		}

		tx := int(math.Floor(mv.X))
		ty := int(math.Floor(mv.Y))

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

func main() {
	plugin.Run(&labTrail{})
}
