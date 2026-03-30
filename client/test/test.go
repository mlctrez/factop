package main

import (
	"fmt"

	"github.com/mlctrez/factop/client"
	"github.com/mlctrez/factop/client/tiles"
)

func main() {
	cl, err := client.Dial("nats://factorio")
	if err != nil {
		panic(err)
	}
	defer cl.Close()
	c := tiles.New(cl)

	a := tiles.Area{X1: -5, Y1: -5, X2: 5, Y2: 5}
	fill, err := c.Fill(a, "concrete", "")
	if err != nil {
		panic(err)
	}
	fmt.Println(fill)

	read, err := c.Read(a, "concrete", "")
	if err != nil {
		panic(err)
	}
	fmt.Println(read)

	remove, err := c.Remove(a, "concrete", "")
	if err != nil {
		panic(err)
	}
	fmt.Println(remove)

}
