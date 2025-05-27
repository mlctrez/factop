package tests

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRcon(t *testing.T) {
	pushSoftModFile(t, "rcon")
	t.Run("ping", testCase(func(t *testing.T, c *testContext) {
		assert.Equal(t, `{"input":"pong"}`, c.rconSc(`factop_rcon.ping("pong")`))
	}))
	t.Run("players", testCase(func(t *testing.T, c *testContext) {
		players := &Players{}
		err := json.Unmarshal([]byte(c.rconSc(`factop_rcon.players()`)), players)
		assert.NoError(t, err)
		for _, player := range players.Players {
			fmt.Println(player.Name)
		}
	}))
	t.Run("game", testCase(func(t *testing.T, c *testContext) {
		game := &Game{}
		err := json.Unmarshal([]byte(c.rconSc(`factop_rcon.game()`)), game)
		assert.NoError(t, err)
		assert.Equal(t, 1, game.Speed)
		assert.Equal(t, false, game.TickPaused, "tick_paused should not be true during rcon")

	}))
}
func TestRemoveOffline(t *testing.T) {
	t.Run("remove_offline", testCase(func(t *testing.T, c *testContext) {
		paramErr := "players is required and must be a table"
		assert.Equal(t, paramErr,
			c.errorString(c.rconSc(`factop_rcon.remove_offline()`)))
		assert.Equal(t, paramErr,
			c.errorString(c.rconSc(`factop_rcon.remove_offline("not_a_table")`)))
		assert.Equal(t, "table must be array of string",
			c.errorString(c.rconSc(`factop_rcon.remove_offline({bad="parameter"})`)))
		assert.Equal(t, "Invalid PlayerIdentification. Player does not exist.",
			c.errorString(c.rconSc(`factop_rcon.remove_offline({"test_player_name_to_remove_bad"})`)))

		result := c.rconSc(`factop_rcon.remove_offline({"test_player_name_to_remove_good"})`)
		players := &Players{}
		assert.NoError(t, json.Unmarshal([]byte(result), players))
		assert.Equal(t, 1, len(players.Players), "incorrect number removed")
		assert.Equal(t, "test_player_name_to_remove_good", players.Players[0].Name, "incorrect name")

		result = c.rconSc(`factop_rcon.remove_offline({})`)
		players = &Players{}
		assert.NoError(t, json.Unmarshal([]byte(result), players))
		assert.Equal(t, 0, len(players.Players), "incorrect number removed")

		result = c.rconSc(`factop_rcon.players()`)
		players = &Players{}
		assert.NoError(t, json.Unmarshal([]byte(result), players))
		for _, player := range players.Players {
			if player.Name == "mlctrez" {
				result = c.rconSc(`factop_rcon.remove_offline({"mlctrez"})`)
				players = &Players{}
				assert.NoError(t, json.Unmarshal([]byte(result), players))
				assert.Equal(t, 1, len(players.Players), "incorrect number removed")
			}
		}
	}))
}

type Players struct {
	Players []struct {
		Index     int    `json:"index"`
		Connected bool   `json:"connected"`
		Name      string `json:"name"`
		Admin     bool   `json:"admin"`
		Position  struct {
			Y float64 `json:"y"`
			X float64 `json:"x"`
		} `json:"position"`
		AfkTime    int    `json:"afk_time"`
		OnlineTime int    `json:"online_time"`
		LastOnline int    `json:"last_online"`
		Force      string `json:"force"`
		Color      struct {
			R float64 `json:"r"`
			G float64 `json:"g"`
			B float64 `json:"b"`
			A float64 `json:"a"`
		} `json:"color"`
	} `json:"players"`
}

type Game struct {
	Tick             int  `json:"tick"`
	TicksPlayed      int  `json:"ticks_played"`
	TickPaused       bool `json:"tick_paused"`
	TicksToRun       int  `json:"ticks_to_run"`
	Speed            int  `json:"speed"`
	AutosaveEnabled  bool `json:"autosave_enabled"`
	BlueprintsCount  int  `json:"blueprints_count"`
	ConnectedPlayers int  `json:"connected_players"`
}
