package main

import (
	"fmt"
	"math"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"

	"github.com/mlctrez/factop/client"
	"github.com/mlctrez/factop/client/prototypes"
	"github.com/mlctrez/factop/client/tiles"
	"github.com/nats-io/nats.go"
)

const tileName = "lab-dark-1"
const radius = 3

// movePattern matches: [move] #<index> <name> to {x=<float>, y=<float>}
var movePattern = regexp.MustCompile(
	`\[move\] #\d+ \S+ to \{x=([+-]?\d+\.?\d*), y=([+-]?\d+\.?\d*)\}`,
)

func main() {
	if !prototypes.ValidTile[tileName] {
		fmt.Fprintf(os.Stderr, "invalid tile name: %s\n", tileName)
		os.Exit(1)
	}

	host := os.Getenv("FACTOP_HOST")
	if host == "" {
		host = "factorio"
	}

	conn, err := client.Dial(fmt.Sprintf("nats://%s", host))
	if err != nil {
		fmt.Fprintf(os.Stderr, "dial: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	tc := tiles.New(conn)

	nc, err := nats.Connect(fmt.Sprintf("nats://%s", host))
	if err != nil {
		fmt.Fprintf(os.Stderr, "nats connect: %v\n", err)
		os.Exit(1)
	}
	defer nc.Close()

	fmt.Printf("watching for player movement, placing %s tiles (radius %d)...\n", tileName, radius)

	_, err = nc.Subscribe("udp.incoming", func(msg *nats.Msg) {
		matches := movePattern.FindSubmatch(msg.Data)
		if matches == nil {
			return
		}
		x, _ := strconv.ParseFloat(string(matches[1]), 64)
		y, _ := strconv.ParseFloat(string(matches[2]), 64)

		// Snap to tile coordinates and build area around player
		tx := int(math.Floor(x))
		ty := int(math.Floor(y))
		area := tiles.Area{
			X1: tx - radius,
			Y1: ty - radius,
			X2: tx + radius + 1,
			Y2: ty + radius + 1,
		}

		result, fillErr := tc.Fill(area, tileName, "")
		if fillErr != nil {
			fmt.Fprintf(os.Stderr, "fill error: %v\n", fillErr)
			return
		}
		fmt.Printf("player at {%.1f, %.1f} -> %s\n", x, y, result)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "subscribe: %v\n", err)
		os.Exit(1)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	fmt.Println("\nshutting down")
}
