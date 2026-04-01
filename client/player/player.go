// Package player provides a typed Go client for the player manipulation
// commands registered by softmod/factop/player.lua.
package player

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mlctrez/factop/client"
)

// Move represents a parsed player movement UDP message.
type Move struct {
	Name         string
	Index        int
	X            float64
	Y            float64
	SurfaceName  string
	SurfaceIndex int
}

// ParseMove extracts player movement data from a UDP message.
// The expected format is:
//
//	[move] #<index> <name> <name>:<index>:<x>:<y>:<surface_name>:<surface_index>
func ParseMove(msg string) (Move, error) {
	fields := strings.Fields(msg)
	if len(fields) < 4 {
		return Move{}, fmt.Errorf("message too short")
	}
	payload := fields[len(fields)-1]
	parts := strings.Split(payload, ":")
	if len(parts) != 6 {
		return Move{}, fmt.Errorf("expected 6 colon-separated fields, got %d", len(parts))
	}
	idx, err := strconv.Atoi(parts[1])
	if err != nil {
		return Move{}, fmt.Errorf("parse index: %w", err)
	}
	x, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return Move{}, fmt.Errorf("parse x: %w", err)
	}
	y, err := strconv.ParseFloat(parts[3], 64)
	if err != nil {
		return Move{}, fmt.Errorf("parse y: %w", err)
	}
	si, err := strconv.Atoi(parts[5])
	if err != nil {
		return Move{}, fmt.Errorf("parse surface index: %w", err)
	}
	return Move{
		Name:         parts[0],
		Index:        idx,
		X:            x,
		Y:            y,
		SurfaceName:  parts[4],
		SurfaceIndex: si,
	}, nil
}

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
