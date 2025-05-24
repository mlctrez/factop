package dscnv

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/mlctrez/factop/api"
	"github.com/nats-io/nats.go"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"time"
)

type RootStructure struct {
	Version int        `json:"version"`
	Data    DataStruct `json:"data"`
}

type DataStruct struct {
	Geometry map[string]GeometryData `json:"geometry"`
	Assets   interface{}             `json:"assets"`
}

type GeometryData struct {
	Polygons  [][][][]float64 `json:"polygons"`
	Polylines []interface{}   `json:"polylines"`
}

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func (pp *Point) String() string {
	return fmt.Sprintf("(%v,%v)", pp.X, pp.Y)
}

type Polygon struct {
	Points []*Point `json:"points"`
	Min    *Point   `json:"min"`
	Max    *Point   `json:"max"`
}

type Dungeon struct {
	Min      *Point `json:"min"`
	Max      *Point `json:"max"`
	Polygons []*Polygon
}

func (d *Dungeon) Dimensions() (width, height int) {
	width = int(d.Max.X - d.Min.X)
	height = int(d.Max.Y - d.Min.Y)
	return width, height
}

func parsePolygons(geometry map[string]GeometryData) *Dungeon {
	d := &Dungeon{}
	d.Min = &Point{X: math.MaxFloat64, Y: math.MaxFloat64}
	d.Max = &Point{X: -math.MaxFloat64, Y: -math.MaxFloat64}

	for _, geom := range geometry {
		for _, polygon := range geom.Polygons {
			for _, ring := range polygon {
				poly := &Polygon{Points: make([]*Point, len(ring))}
				for i, point := range ring {
					x := math.RoundToEven(point[0])
					y := math.RoundToEven(point[1])
					d.Min.X = math.Min(d.Min.X, x)
					d.Min.Y = math.Min(d.Min.Y, y)
					d.Max.X = math.Max(d.Max.X, x)
					d.Max.Y = math.Max(d.Max.Y, y)
					poly.Points[i] = &Point{X: x, Y: y}
				}
				d.Polygons = append(d.Polygons, poly)
			}
		}
	}
	return d
}

//go:embed dungeon.ds
var dungeon []byte

func readDungeonMap() (*RootStructure, error) {
	reader, err := zip.NewReader(bytes.NewReader(dungeon), int64(len(dungeon)))
	if err != nil {
		return nil, err
	}

	for _, file := range reader.File {
		if file.Name == "map" {
			var open io.ReadCloser
			if open, err = file.Open(); err != nil {
				return nil, err
			}
			var mapJson []byte
			if mapJson, err = io.ReadAll(open); err != nil {
				return nil, err
			}
			root := &RootStructure{}
			if err = json.Unmarshal(mapJson, root); err != nil {
				return nil, err
			}
			return root, nil
		}
	}

	return nil, fmt.Errorf("no dungeon found")
}

var natsConn *nats.Conn
var client *api.RconClient

func Run() {

	exec.Command("mage", "rcon", "example/remove_walls.lua").Run()
	exec.Command("mage", "rcon", "softmod/factop/wall.lua").Run()

	log.Default().SetFlags(0)

	con, err := nats.Connect("nats://factorio")
	if err != nil {
		log.Fatal(err)
	}
	natsConn = con
	defer natsConn.Close()

	client = api.NewRconClient(con)

	root, err := readDungeonMap()
	if err != nil {
		log.Fatal(err)
	}
	dun := parsePolygons(root.Data.Geometry)
	//fmt.Printf("Min: %+v  Max:%+v\n", dun.Min, dun.Max)
	//fmt.Printf("Polygons: %d\n", len(dun.Polygons))
	width, height := dun.Dimensions()
	//fmt.Printf("Dimensions: %d %d\n", width, height)
	border := 20
	borderFloat := float64(border)
	img := image.NewRGBA(image.Rect(0, 0, width+border*2, height+border*2))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.Black}, image.Point{}, draw.Src)

	for _, polygon := range dun.Polygons {
		numPoints := len(polygon.Points) - 1
		for i := 0; i < numPoints; i++ {
			start := polygon.Points[i]
			end := polygon.Points[i+1]
			//fmt.Printf("start: %s -> %s\n", start, end)
			actualStart := &Point{X: start.X - dun.Min.X + borderFloat, Y: start.Y - dun.Min.Y + borderFloat}
			actualEnd := &Point{X: end.X - dun.Min.X + borderFloat, Y: end.Y - dun.Min.Y + borderFloat}
			col := color.RGBA{R: 255, G: 255, B: 255, A: 255}
			if i == 0 {
				//col = color.RGBA{R: 255, A: 255}
			}
			drawLine(img, actualStart, actualEnd, 2, col)
		}
	}
	f, err := os.Create("polygons.png")
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		fmt.Printf("Error encoding PNG: %v\n", err)
		return
	}

	fmt.Printf("Image saved as polygons.png (%dx%d pixels)\n", img.Bounds().Max.X, img.Bounds().Max.Y)

}

func drawLine(img *image.RGBA, start, end *Point, width float64, c color.Color) {
	dx := end.X - start.X
	dy := end.Y - start.Y
	length := math.Sqrt(dx*dx + dy*dy)
	if length == 0 {
		return
	}

	dx /= length
	dy /= length
	px := -dy
	py := dx

	var w float64
	var t float64

	for w = -width; w <= width; w = w + 0.5 {
		for t = 0; t <= length; t = t + 0.5 {
			x := int(start.X + t*dx + w*px)
			y := int(start.Y + t*dy + w*py)
			if x >= 0 && x < img.Bounds().Max.X && y >= 0 && y < img.Bounds().Max.Y {
				img.Set(x, y, c)
			}
		}
	}

	scale := 0.5
	offset := &Point{X: -17, Y: -17}

	//payload := fmt.Sprintf(`/sc factop_wall.wall_line("nauvis", "player", {x=%f,y=%f}, {x=%f,y=%f}, 1)`,
	//	offset.X+start.X*scale, offset.Y+start.Y*scale, offset.X+end.X*scale, offset.Y+end.Y*scale)
	payload := fmt.Sprintf(`/sc factop_tiles.tile_line("nauvis", "refined-concrete", {x=%f,y=%f}, {x=%f,y=%f}, 2)`,
		offset.X+start.X*scale, offset.Y+start.Y*scale, offset.X+end.X*scale, offset.Y+end.Y*scale)

	result, err := client.Execute(&api.RconCommand{Payload: payload})
	if err != nil {
		log.Fatal(err)
	}
	if result.Payload != "" {
		fmt.Println(result.Payload)
	}
	time.Sleep(50 * time.Millisecond)
}
