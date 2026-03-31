// Package game provides a typed Go client for the server administration
// commands registered by softmod/factop/game.lua.
package game

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mlctrez/factop/client"
)

// Player represents a connected player from game-players.
type Player struct {
	Name         string  `json:"name"`
	Index        int     `json:"index"`
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	HasCharacter bool    `json:"has_character"`
}

// PlayerEntry represents any player (connected or disconnected) from game-players-all.
type PlayerEntry struct {
	Name      string `json:"name"`
	Index     int    `json:"index"`
	Connected bool   `json:"connected"`
}

// Client provides typed methods for each game-* RCON command.
type Client struct {
	conn *client.Conn
}

// New creates a game Client using the given connection.
func New(conn *client.Conn) *Client {
	return &Client{conn: conn}
}

// Players returns all connected players.
func (c *Client) Players() ([]Player, error) {
	raw, err := c.conn.Rcon("/game-players")
	if err != nil {
		return nil, err
	}
	return ParsePlayers(raw)
}

// PlayersAll returns all players (connected and disconnected).
func (c *Client) PlayersAll() ([]PlayerEntry, error) {
	raw, err := c.conn.Rcon("/game-players-all")
	if err != nil {
		return nil, err
	}
	return ParsePlayersAll(raw)
}

// Kick disconnects a player from the server.
func (c *Client) Kick(player string, reason string) (string, error) {
	cmd := fmt.Sprintf("/game-kick %s", player)
	if reason != "" {
		cmd += " " + reason
	}
	return c.conn.Rcon(cmd)
}

// Remove kicks a player (if connected) and removes them from the save.
// On reconnect they will start as a new player.
func (c *Client) Remove(player string) (string, error) {
	return c.conn.Rcon(fmt.Sprintf("/game-remove %s", player))
}

// ParsePlayers converts the game-players wire format into a slice of Player.
// Wire format: name:index:x:y:has_character,...
func ParsePlayers(s string) ([]Player, error) {
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

// ParsePlayersAll converts the game-players-all wire format into a slice of PlayerEntry.
// Wire format: name:index:connected,...
func ParsePlayersAll(s string) ([]PlayerEntry, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	entries := strings.Split(s, ",")
	result := make([]PlayerEntry, 0, len(entries))
	for _, entry := range entries {
		parts := strings.SplitN(entry, ":", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid player-all entry %q: expected name:index:connected", entry)
		}
		idx, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid index in %q: %w", entry, err)
		}
		result = append(result, PlayerEntry{
			Name:      parts[0],
			Index:     idx,
			Connected: parts[2] == "true",
		})
	}
	return result, nil
}
