// Package players provides a typed Go client for the player manipulation
// commands registered by softmod/factop/players.lua.
package player

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mlctrez/factop/client"
)

// Player represents a connected player from players-list.
type Player struct {
	Name         string  `json:"name"`
	Index        int     `json:"index"`
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	HasCharacter bool    `json:"has_character"`
}

// Client provides typed methods for each players-* RCON command.
type Client struct {
	conn *client.Conn
}

// New creates a players Client using the given connection.
func New(conn *client.Conn) *Client {
	return &Client{conn: conn}
}

// List returns all connected players.
func (c *Client) List() ([]Player, error) {
	raw, err := c.conn.Rcon("/players-list")
	if err != nil {
		return nil, err
	}
	return Parse(raw)
}

// Respawn creates a new character for a player without one.
// Pass empty string to respawn all connected players without characters.
func (c *Client) Respawn(player string) (string, error) {
	cmd := "/players-respawn"
	if player != "" {
		cmd += " " + player
	}
	return c.conn.Rcon(cmd)
}

// Teleport moves a player to the given position.
func (c *Client) Teleport(player string, x, y float64, surface string) (string, error) {
	cmd := fmt.Sprintf("/players-teleport %s %g,%g", player, x, y)
	if surface != "" {
		cmd += " " + surface
	}
	return c.conn.Rcon(cmd)
}

// Parse converts the players-list wire format into a slice of Player.
// Wire format: name:index:x:y:has_character,...
func Parse(s string) ([]Player, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	entries := strings.Split(s, ",")
	result := make([]Player, 0, len(entries))
	for _, entry := range entries {
		parts := strings.SplitN(entry, ":", 5)
		if len(parts) != 5 {
			return nil, fmt.Errorf("invalid player entry %q: expected name:index:x:y:has_character", entry)
		}
		idx, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid index in %q: %w", entry, err)
		}
		x, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid x in %q: %w", entry, err)
		}
		y, err := strconv.ParseFloat(parts[3], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid y in %q: %w", entry, err)
		}
		result = append(result, Player{
			Name:         parts[0],
			Index:        idx,
			X:            x,
			Y:            y,
			HasCharacter: parts[4] == "true",
		})
	}
	return result, nil
}
