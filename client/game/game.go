package game

// gen:event tag=game_tick
// gen:lua event=on_tick:game_tick
// gen:lua tag=game_tick
// gen:lua nth_tick=60
// gen:lua field.Tick=game.tick
type TickEvent struct {
	Tick uint64 `wire:"0"`
}
