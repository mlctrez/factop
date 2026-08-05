package surface

// gen:event tag=surface_created
// gen:event tag=surface_deleted
// gen:lua event=on_surface_created:surface_created
// gen:lua event=on_surface_deleted:surface_deleted
// gen:lua tag=surface_created
// gen:lua guard=game.surfaces[event.surface_index]
// gen:lua field.Name=surface.name
// gen:lua field.Index=surface.index
// gen:lua fallback.object=surface
// gen:lua fallback.default.Name=""
// gen:lua fallback.default.Index=event.surface_index
type SurfaceEvent struct {
	Event string `wire:"-,tag"`
	Name  string `wire:"0"`
	Index int    `wire:"1"`
}
