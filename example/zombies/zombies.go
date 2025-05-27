package main

import (
	_ "embed"
	"fmt"
	"github.com/mlctrez/factop/example"
	"os"
	"os/signal"
	"syscall"
	"time"
)

//go:embed zombies.lua
var script string

func callScript(ex *example.Example, method string) {
	ex.RconSc(script + fmt.Sprintf("\nzombies.%s()", method))
}

func main() {
	ex := example.New()
	defer ex.Close()

	spawner := time.NewTicker(200 * time.Millisecond)
	mover := time.NewTicker(20 * time.Millisecond)
	clearer := time.NewTicker(30 * time.Second)
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	defer spawner.Stop()
	defer clearer.Stop()

	for {
		select {
		case <-spawner.C:
			callScript(ex, "create")
		case <-mover.C:
			callScript(ex, "move")
		case <-clearer.C:
			callScript(ex, "delete")

		case <-done:
			callScript(ex, "delete")
			return
		}
	}

}
