package main

import (
	"fmt"
	"math"
	"math/rand/v2"

	"github.com/mlctrez/factop/client/player"
	"github.com/mlctrez/factop/plugin"
)

var labTiles = []string{"lab-dark-1", "lab-dark-2", "lab-white"}

type labTrail struct{}

func (l *labTrail) Setup(ctx plugin.Context) error {
	ctx.Logger().Info("setup complete")
	return nil
}

func (l *labTrail) Run(ctx plugin.Context) error {
	log := ctx.Logger()

	err := ctx.Events().OnMove(func(mv player.Move) {
		tx := int(math.Floor(mv.X))
		ty := int(math.Floor(mv.Y))
		name := labTiles[rand.IntN(len(labTiles))]
		// Direct RCON call for tile placement until Fire() is available.
		cmd := fmt.Sprintf("/sc game.surfaces['%s'].set_tiles({{name='%s', position={%d,%d}}})",
			mv.SurfaceName, name, tx, ty)
		if _, err := ctx.Rcon(cmd); err != nil {
			log.Error("tile placement failed", "error", err)
		}
	})
	if err != nil {
		return fmt.Errorf("register move handler: %w", err)
	}

	log.Info("running")
	<-ctx.Done()
	log.Info("shutting down")
	return nil
}

func main() {
	plugin.Run(&labTrail{})
}
