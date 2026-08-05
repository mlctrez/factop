package tile

// gen:event tag=tile_built
// gen:event tag=tile_mined
// gen:lua event=on_player_built_tile:tile_built
// gen:lua event=on_robot_built_tile:tile_built
// gen:lua event=on_player_mined_tile:tile_mined
// gen:lua event=on_robot_mined_tile:tile_mined
// gen:lua tag=tile_built
// gen:lua guard=game.surfaces[event.surface_index]
// gen:lua iterate=event.tiles
// gen:lua loop_var=tile
// gen:lua field.Name=tile.old_tile.name
// gen:lua field.X=tile.position.x
// gen:lua field.Y=tile.position.y
// gen:lua field.SurfaceName=surface.name
// gen:lua field.SurfaceIndex=surface.index
// gen:lua field.PlayerIndex=event.player_index
// gen:lua default.PlayerIndex=0
type TileEvent struct {
	Event        string  `wire:"-,tag"`
	Name         string  `wire:"0"`
	X            float64 `wire:"1"`
	Y            float64 `wire:"2"`
	SurfaceName  string  `wire:"3"`
	SurfaceIndex int     `wire:"4"`
	PlayerIndex  int     `wire:"5"`
}
