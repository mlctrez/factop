package main

import (
	"fmt"
	"github.com/mlctrez/factop/example"
	"github.com/ojrac/opensimplex-go"
	"time"
)

var rc = example.New()

func main() {
	defer rc.Close()

	width, height := 512, 512

	// Noise parameters
	seed := int64(23452345)
	scale := 0.01 // Controls the "zoom" of the noise

	openNoise := opensimplex.New(seed)

	for yo := 0; yo < height; yo++ {
		fmt.Println(yo)
		row := make([]string, width)
		for xo := 0; xo < width; xo++ {
			noise := openNoise.Eval2(float64(xo)*scale, float64(yo)*scale)
			if noise < -0.2 {
				row[xo] = "water"
			} else if noise < 0.1 {
				row[xo] = "sand-1"
			} else if noise < 0.3 {
				row[xo] = "dirt-1"
			} else {
				row[xo] = "grass-1"
			}
		}
		findAndProcessSegments(yo, row)
	}
}

func findAndProcessSegments(y int, arr []string) {
	currentValue := arr[0]
	start := 0

	for i, value := range arr {
		if value != currentValue || i == len(arr)-1 {
			if value != currentValue {
				end := i - 1
				processSegment(y, currentValue, start, end)
				currentValue = value
				start = i
			}
			if i == len(arr)-1 {
				end := i
				processSegment(y, currentValue, start, end)
			}
		}
	}
}

func processSegment(y int, value string, start, end int) {
	sx, ex := start-256, end-256
	sy, ey := y-256, y-256
	cmd := fmt.Sprintf("factop_tiles.create(%d,%d,%d,%d,\"lab\",%q)", sx, ex, sy, ey, value)
	//fmt.Println(cmd)
	rc.RconSc(cmd)
	time.Sleep(10 * time.Millisecond)
}
