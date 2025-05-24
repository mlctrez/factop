package main

import (
	"github.com/mlctrez/factop/example/dyntile"
	"log"
)

func main() {
	log.SetFlags(0)
	err := dyntile.Run()
	if err != nil {
		log.Fatal(err)
	}
}
