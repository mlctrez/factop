package playerattr

// gen:event tag=player_joined
// gen:event tag=player_left
// gen:event tag=player_died
// gen:event tag=player_respawned
// gen:lua event=on_player_joined_game:player_joined
// gen:lua event=on_player_left_game:player_left
// gen:lua event=on_player_died:player_died
// gen:lua event=on_player_respawned:player_respawned
// gen:lua tag=player_joined
// gen:lua guard=game.players[event.player_index]
// gen:lua field.Name=player.name
// gen:lua field.Index=player.index
type PlayerEvent struct {
	Event string `wire:"-,tag"`
	Name  string `wire:"0"`
	Index int    `wire:"1"`
}
