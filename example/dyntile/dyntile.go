package dyntile

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/mlctrez/factop/api"
	"github.com/nats-io/nats.go"
	"image"
	_ "image/png" // register PNG format
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
)

//go:embed nordschleife.png
var nordschleife []byte

// FilteredMap stores the parsed image globally
var nordschleifeImg image.Image

func Run() error {
	var err error
	nordschleifeImg, _, err = image.Decode(bytes.NewReader(nordschleife))
	if err != nil {
		return fmt.Errorf("failed to decode filtered_map.png: %v", err)
	}

	connect, err := nats.Connect("nats://factorio")
	if err != nil {
		return err
	}
	defer connect.Close()

	apiClient := api.NewRconClient(connect)

	sub, err := connect.Subscribe("factorio.stdout", func(msg *nats.Msg) {
		// factop_chunk.generated {x = 7, y = 7}
		data := string(msg.Data)
		if strings.Contains(data, "factop_chunk.generated") {
			var x, y int
			_, err := fmt.Sscanf(data, "factop_chunk.generated {x = %d, y = %d}", &x, &y)
			if err != nil {
				fmt.Printf("Error parsing coordinates: %v\n", err)
				return
			}
			handleChunkGenerated(apiClient, x, y)
		}
	})
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	return nil
}

func handleChunkGenerated(client *api.RconClient, chunkX int, chunkY int) {
	var pixels []struct{ x, y int }

	chunkSize := 32
	startX := chunkX * chunkSize
	startY := chunkY * chunkSize
	endX := startX + chunkSize - 1
	endY := startY + chunkSize - 1

	bounds := nordschleifeImg.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()

	imgCenterOffset := imgHeight / 2

	for y := startY; y <= endY; y++ {
		for x := startX; x <= endX; x++ {
			imgX := x
			imgY := y + imgCenterOffset

			if imgX >= 0 && imgX < imgWidth && imgY >= 0 && imgY < imgHeight {
				c := nordschleifeImg.At(imgX, imgY)
				_, _, _, a := c.RGBA()
				// include all pixels that are not transparent
				if a != 0 {
					pixels = append(pixels, struct{ x, y int }{x: x, y: y})
				}
			}
		}
	}

	sort.Slice(pixels, func(i, j int) bool {
		if pixels[i].y != pixels[j].y {
			return pixels[i].y < pixels[j].y
		}
		return pixels[i].x < pixels[j].x
	})

	var currentLine struct {
		startX, endX, y int
		active          bool
	}

	var createTiles = func(surface, name string, sx, sy, ex, ey int) {
		payload := fmt.Sprintf("/sc factop_tiles.create(%d,%d,%d,%d,%q,%q)", sx, ex, sy, ey, surface, name)
		_, err := client.Execute(&api.RconCommand{Payload: payload})
		if err != nil {
			fmt.Printf("Error creating tiles: %v\n", err)
		}
	}

	for i, pixel := range pixels {
		if !currentLine.active {
			currentLine.startX = pixel.x
			currentLine.y = pixel.y
			currentLine.active = true
		} else if pixel.y != currentLine.y || (i > 0 && pixel.x > pixels[i-1].x+1) {
			currentLine.endX = pixels[i-1].x
			createTiles("nauvis", "concrete", currentLine.startX, currentLine.y, currentLine.endX, currentLine.y)

			currentLine.startX = pixel.x
			currentLine.y = pixel.y
		}

		if i == len(pixels)-1 {
			currentLine.endX = pixel.x
			createTiles("nauvis", "concrete", currentLine.startX, currentLine.y, currentLine.endX, currentLine.y)
		}
	}
}
