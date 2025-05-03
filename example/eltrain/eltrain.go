package main

import (
	_ "embed"
	"github.com/mlctrez/factop/example"
	"time"
)

func main() {
	ex := example.New()
	ex.DontExitOnError = true
	defer ex.Close()
	ex.RconSc(destroyTrain)
	ex.RconSc(destroyRail)
	time.Sleep(10 * time.Second)
	ex.RconSc(createRail)

	for {
		time.Sleep(1 * time.Second)
		for i := 0; i < 5; i++ {
			ex.RconSc(createTrain)
			time.Sleep(400 * time.Millisecond)
		}
		time.Sleep(30 * time.Second)
		ex.RconSc(destroyTrain)
	}

}

//go:embed create_rail.lua
var createRail string

//go:embed create_train.lua
var createTrain string

//go:embed destroy_train.lua
var destroyTrain string

//go:embed destroy_rail.lua
var destroyRail string
