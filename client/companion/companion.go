package companion

// gen:event tag=companion_summoned
// gen:event tag=companion_dismissed
// gen:event tag=companion_died
type CompanionEvent struct {
	Event        string  `wire:"-,tag"`
	Name         string  `wire:"0"`
	PlayerIndex  int     `wire:"1"`
	PlayerName   string  `wire:"2"`
	UnitNumber   int     `wire:"3"`
	X            float64 `wire:"4"`
	Y            float64 `wire:"5"`
	SurfaceName  string  `wire:"6"`
	SurfaceIndex int     `wire:"7"`
}
