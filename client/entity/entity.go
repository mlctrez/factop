package entity

// gen:event tag=entity_died
// gen:event tag=entity_built
// gen:event tag=entity_mined
// gen:lua event=on_entity_died:entity_died
// gen:lua event=on_built_entity:entity_built
// gen:lua event=on_robot_built_entity:entity_built
// gen:lua event=on_player_mined_entity:entity_mined
// gen:lua event=on_robot_mined_entity:entity_mined
// gen:lua tag=entity_died
// gen:lua guard=event.entity
// gen:lua guard=entity.surface
// gen:lua field.Name=entity.name
// gen:lua field.X=entity.position.x
// gen:lua field.Y=entity.position.y
// gen:lua field.UnitNumber=entity.unit_number
// gen:lua field.SurfaceName=surface.name
// gen:lua field.SurfaceIndex=surface.index
// gen:lua field.PlayerIndex=event.player_index
// gen:lua field.Cause=cause
// gen:lua default.UnitNumber=0
// gen:lua default.PlayerIndex=0
// gen:lua default.Cause=""
// gen:lua include_cause=entity_died
type EntityEvent struct {
	Event        string  `wire:"-,tag"`
	Name         string  `wire:"0"`
	X            float64 `wire:"1"`
	Y            float64 `wire:"2"`
	UnitNumber   uint64  `wire:"3"`
	SurfaceName  string  `wire:"4"`
	SurfaceIndex int     `wire:"5"`
	PlayerIndex  int     `wire:"6"`
	Cause        string  `wire:"7,optional"`
}
