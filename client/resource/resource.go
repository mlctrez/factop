package resource

// gen:event tag=resource_depleted
// gen:lua event=on_resource_depleted:resource_depleted
// gen:lua tag=resource_depleted
// gen:lua guard=event.entity
// gen:lua guard=entity.surface
// gen:lua field.Name=entity.name
// gen:lua field.X=entity.position.x
// gen:lua field.Y=entity.position.y
// gen:lua field.Amount=entity.amount
// gen:lua field.SurfaceName=surface.name
// gen:lua field.SurfaceIndex=surface.index
type ResourceDepleted struct {
	Name         string  `wire:"0"`
	X            float64 `wire:"1"`
	Y            float64 `wire:"2"`
	Amount       uint64  `wire:"3"`
	SurfaceName  string  `wire:"4"`
	SurfaceIndex int     `wire:"5"`
}
