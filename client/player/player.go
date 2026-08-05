package player

// gen:event tag=move
type Move struct {
	Name         string  `wire:"0"`
	Index        int     `wire:"1"`
	X            float64 `wire:"2"`
	Y            float64 `wire:"3"`
	SurfaceName  string  `wire:"4"`
	SurfaceIndex int     `wire:"5"`
}
