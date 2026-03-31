package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/mlctrez/factop/client"
	"github.com/mlctrez/factop/client/entity"
	"github.com/mlctrez/factop/client/game"
	"github.com/mlctrez/factop/client/player"
	"github.com/mlctrez/factop/client/prototype"
	"github.com/mlctrez/factop/client/tile"
)

const (
	wallName  = "stone-wall"
	floorTile = "lab-dark-1"
	boxSize   = 10  // inner box dimension
	mazeExt   = 50  // maze extends this far outside the box
	clearSize = 120 // area to clear entities and lay floor tiles
)

// origin centers the structure at 0,0
var originX = -boxSize / 2
var originY = -boxSize / 2

func main() {
	if !prototype.ValidWall[wallName] {
		fmt.Fprintf(os.Stderr, "invalid wall name: %s\n", wallName)
		os.Exit(1)
	}
	if !prototype.ValidTile[floorTile] {
		fmt.Fprintf(os.Stderr, "invalid tile name: %s\n", floorTile)
		os.Exit(1)
	}

	host := os.Getenv("FACTOP_HOST")
	if host == "" {
		host = "factorio"
	}

	conn, err := client.Dial(fmt.Sprintf("nats://%s", host), client.WithTimeout(30*time.Second))
	if err != nil {
		fmt.Fprintf(os.Stderr, "dial: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	tc := tile.New(conn)
	ec := entity.New(conn)
	gc := game.New(conn)
	pc := player.New(conn)

	// Remove all disconnected players except mlctrez
	removeDisconnectedPlayers(gc, "mlctrez")

	// Clear the full area and lay floor
	clearArea := tile.Area{
		X1: -clearSize, Y1: -clearSize,
		X2: clearSize, Y2: clearSize,
	}
	fmt.Println("clearing area and laying floor...")
	destroyArea := entity.Area{
		X1: float64(clearArea.X1), Y1: float64(clearArea.Y1),
		X2: float64(clearArea.X2), Y2: float64(clearArea.Y2),
	}
	clearEntities(ec, destroyArea)
	_, _ = tc.Fill(clearArea, floorTile, "")

	// Respawn any players that lost their character during clearing
	respawnPlayers(gc, pc)

	// Build the 50x50 box walls
	fmt.Println("building box walls...")
	placed := 0
	for i := 0; i < boxSize; i++ {
		// Top wall
		placeWall(ec, originX+i, originY-1, &placed)
		// Bottom wall
		placeWall(ec, originX+i, originY+boxSize, &placed)
		// Left wall
		placeWall(ec, originX-1, originY+i, &placed)
		// Right wall
		placeWall(ec, originX+boxSize, originY+i, &placed)
	}
	// Corners
	placeWall(ec, originX-1, originY-1, &placed)
	placeWall(ec, originX+boxSize, originY-1, &placed)
	placeWall(ec, originX-1, originY+boxSize, &placed)
	placeWall(ec, originX+boxSize, originY+boxSize, &placed)
	fmt.Printf("box complete: %d walls placed\n", placed)

	// Generate maze outside the box
	fmt.Println("generating maze...")
	mazeWalls := generateMaze()
	fmt.Printf("placing %d maze walls...\n", len(mazeWalls))
	for _, p := range mazeWalls {
		placeWall(ec, p[0], p[1], &placed)
	}
	fmt.Printf("done: %d total walls placed\n", placed)
}

// removeDisconnectedPlayers removes all players from the save except the
// given keep name. Only disconnected players are removed.
func removeDisconnectedPlayers(gc *game.Client, keep string) {
	all, err := gc.PlayersAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "list all players: %v\n", err)
		return
	}
	for _, p := range all {
		if p.Name == keep {
			continue
		}
		if !p.Connected {
			fmt.Printf("  removing disconnected player %s (index %d)...\n", p.Name, p.Index)
			result, rErr := gc.Remove(p.Name)
			if rErr != nil {
				fmt.Fprintf(os.Stderr, "  remove %s: %v\n", p.Name, rErr)
			} else {
				fmt.Printf("  %s\n", result)
			}
		}
	}
}

// respawnPlayers checks all connected players and creates characters for
// any that are missing one.
func respawnPlayers(gc *game.Client, pc *player.Client) {
	list, err := gc.Players()
	if err != nil {
		fmt.Fprintf(os.Stderr, "list players: %v\n", err)
		return
	}
	for _, p := range list {
		if !p.HasCharacter {
			fmt.Printf("  respawning %s (index %d)...\n", p.Name, p.Index)
			result, rErr := pc.Respawn(p.Name)
			if rErr != nil {
				fmt.Fprintf(os.Stderr, "  respawn %s: %v\n", p.Name, rErr)
			} else {
				fmt.Printf("  %s\n", result)
			}
		}
	}
}

// protectedTypes are entity types that should never be destroyed during clearing.
var protectedTypes = map[string]bool{
	"character":        true,
	"character-corpse": true,
}

// clearEntities destroys all entities in the area except player characters.
func clearEntities(ec *entity.Client, area entity.Area) {
	found, err := ec.Find(area, entity.FindOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "find for clear: %v\n", err)
		return
	}

	// Collect unique entity names, skipping protected types
	names := make(map[string]bool)
	for _, e := range found {
		if !protectedTypes[e.Name] {
			names[e.Name] = true
		}
	}

	for name := range names {
		result, dErr := ec.Destroy(area, entity.FindOptions{Name: name})
		if dErr != nil {
			fmt.Fprintf(os.Stderr, "destroy %s: %v\n", name, dErr)
			continue
		}
		fmt.Printf("  cleared %s: %s\n", name, result)
	}
}

func placeWall(ec *entity.Client, x, y int, count *int) {
	pos := entity.Position{X: float64(x) + 0.5, Y: float64(y) + 0.5}
	_, err := ec.Create(pos, wallName, "player", "", "")
	if err != nil {
		// Walls can fail if something is already there — not fatal
		fmt.Fprintf(os.Stderr, "wall at %d,%d: %v\n", x, y, err)
		return
	}
	*count++
	if *count%100 == 0 {
		fmt.Printf("  placed %d walls...\n", *count)
	}
}

// generateMaze creates a maze in the 10-tile border around the box using
// recursive backtracker. The maze grid uses 2-cell spacing (wall + passage)
// so corridors are 1 tile wide with walls between them.
func generateMaze() [][2]int {
	// Maze dimensions in cells (each cell = 2 tiles: one passage, one wall)
	// We need mazeExt tiles of maze on each side, so cells = mazeExt/2
	cellsW := mazeExt / 2
	totalW := boxSize + 2 + 2*mazeExt // full width including box border
	totalH := boxSize + 2 + 2*mazeExt

	// The maze occupies the border ring: mazeExt tiles outside the box walls.
	// We generate four strips (top, bottom, left, right) plus corners.
	// Simpler approach: generate a full grid covering the border, mark box
	// interior as excluded.

	gridW := totalW
	gridH := totalH
	// Grid origin in world coords
	gx0 := originX - mazeExt - 1
	gy0 := originY - mazeExt - 1

	// Cell grid for maze generation (half resolution)
	cw := gridW / 2
	ch := gridH / 2
	visited := make([][]bool, ch)
	for i := range visited {
		visited[i] = make([]bool, cw)
	}

	// Mark cells that overlap the box interior as visited (excluded)
	for cy := 0; cy < ch; cy++ {
		for cx := 0; cx < cw; cx++ {
			wx := gx0 + cx*2
			wy := gy0 + cy*2
			// Inside the box walls (not the walls themselves)
			if wx >= originX && wx < originX+boxSize && wy >= originY && wy < originY+boxSize {
				visited[cy][cx] = true
			}
		}
	}

	// Walls: start with everything as wall, carve passages
	isWall := make([][]bool, gridH)
	for i := range isWall {
		isWall[i] = make([]bool, gridW)
		for j := range isWall[i] {
			// Default to wall in the maze border, empty inside box
			wx := gx0 + j
			wy := gx0 + i
			if wx >= originX && wx < originX+boxSize && wy >= originY && wy < originY+boxSize {
				isWall[i][j] = false
			} else {
				isWall[i][j] = true
			}
		}
	}

	// Find a starting cell in the maze border
	startCX, startCY := -1, -1
	for cy := 0; cy < ch && startCX == -1; cy++ {
		for cx := 0; cx < cw; cx++ {
			if !visited[cy][cx] {
				startCX, startCY = cx, cy
				break
			}
		}
	}

	if startCX == -1 {
		return nil
	}

	_ = cellsW // used for documentation

	// Recursive backtracker
	type cell struct{ x, y int }
	stack := []cell{{startCX, startCY}}
	visited[startCY][startCX] = true
	// Carve the starting cell
	isWall[startCY*2][startCX*2] = false

	dirs := [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}

	for len(stack) > 0 {
		cur := stack[len(stack)-1]

		// Find unvisited neighbors
		var neighbors []cell
		for _, d := range dirs {
			nx, ny := cur.x+d[0], cur.y+d[1]
			if nx >= 0 && nx < cw && ny >= 0 && ny < ch && !visited[ny][nx] {
				neighbors = append(neighbors, cell{nx, ny})
			}
		}

		if len(neighbors) == 0 {
			stack = stack[:len(stack)-1]
			continue
		}

		// Pick random neighbor
		next := neighbors[rand.Intn(len(neighbors))]
		visited[next.y][next.x] = true

		// Carve passage: clear the cell and the wall between
		isWall[next.y*2][next.x*2] = false
		wallY := cur.y*2 + (next.y - cur.y)
		wallX := cur.x*2 + (next.x - cur.x)
		if wallY >= 0 && wallY < gridH && wallX >= 0 && wallX < gridW {
			isWall[wallY][wallX] = false
		}

		stack = append(stack, next)
	}

	// Collect wall positions, excluding the box interior and box walls
	// (those are already placed)
	var walls [][2]int
	for gy := 0; gy < gridH; gy++ {
		for gx := 0; gx < gridW; gx++ {
			if !isWall[gy][gx] {
				continue
			}
			wx := gx0 + gx
			wy := gy0 + gy
			// Skip anything inside or on the box wall line (already placed)
			if wx >= originX-1 && wx <= originX+boxSize && wy >= originY-1 && wy <= originY+boxSize {
				continue
			}
			walls = append(walls, [2]int{wx, wy})
		}
	}

	return walls
}
