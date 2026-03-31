// Package player provides a typed Go client for the player manipulation
// commands registered by softmod/factop/player.lua.
package player

import (
	"fmt"

	"github.com/mlctrez/factop/client"
)

// Client provides typed methods for each player-* RCON command.
type Client struct {
	conn *client.Conn
}

// New creates a player Client using the given connection.
func New(conn *client.Conn) *Client {
	return &Client{conn: conn}
}

// Respawn creates a new character for a player without one.
// Pass empty string to respawn all connected players without characters.
func (c *Client) Respawn(player string) (string, error) {
	cmd := "/player-respawn"
	if player != "" {
		cmd += " " + player
	}
	return c.conn.Rcon(cmd)
}

// Teleport moves a player to the given position.
func (c *Client) Teleport(player string, x, y float64, surface string) (string, error) {
	cmd := fmt.Sprintf("/player-teleport %s %g,%g", player, x, y)
	if surface != "" {
		cmd += " " + surface
	}
	return c.conn.Rcon(cmd)
}
