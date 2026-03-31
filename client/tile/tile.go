// Package tiles provides a typed Go client for the tile manipulation commands
// registered by softmod/factop/tiles.lua.
package tile

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mlctrez/factop/client"
)

// Tile represents a single tile at a position.
type Tile struct {
	Name string `json:"name"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
}

// Area defines a rectangular region in tile coordinates.
type Area struct {
	X1 int
	Y1 int
	X2 int
	Y2 int
}

// String formats the area as the x1,y1,x2,y2 wire format.
func (a Area) String() string {
	return fmt.Sprintf("%d,%d,%d,%d", a.X1, a.Y1, a.X2, a.Y2)
}

// Client provides typed methods for each tiles-* RCON command.
type Client struct {
	conn *client.Conn
}

// New creates a tiles Client using the given connection.
func New(conn *client.Conn) *Client {
	return &Client{conn: conn}
}

// Fill places a single tile type across the entire area.
// Corresponds to: /tiles-fill x1,y1,x2,y2 tileName [surface]
func (c *Client) Fill(area Area, tileName string, surface string) (string, error) {
	cmd := fmt.Sprintf("/tiles-fill %s %s", area, tileName)
	if surface != "" {
		cmd += " " + surface
	}
	return c.conn.Rcon(cmd)
}

// Read returns the tiles in the given area, optionally filtered by name.
// Corresponds to: /tiles-read x1,y1,x2,y2 [filterName] [surface]
func (c *Client) Read(area Area, filterName string, surface string) ([]Tile, error) {
	cmd := fmt.Sprintf("/tiles-read %s", area)
	if filterName != "" {
		cmd += " " + filterName
	}
	if surface != "" {
		if filterName == "" {
			// need a placeholder so surface lands in the right arg position
			cmd += " \"\""
		}
		cmd += " " + surface
	}
	raw, err := c.conn.Rcon(cmd)
	if err != nil {
		return nil, err
	}
	return Parse(raw)
}

// Remove restores hidden tiles in the area, optionally filtered by name.
// Corresponds to: /tiles-remove x1,y1,x2,y2 [filterName] [surface]
func (c *Client) Remove(area Area, filterName string, surface string) (string, error) {
	cmd := fmt.Sprintf("/tiles-remove %s", area)
	if filterName != "" {
		cmd += " " + filterName
	}
	if surface != "" {
		if filterName == "" {
			cmd += " \"\""
		}
		cmd += " " + surface
	}
	return c.conn.Rcon(cmd)
}

// Replace swaps one tile type for another across the area.
// Corresponds to: /tiles-replace x1,y1,x2,y2 fromName toName [surface]
func (c *Client) Replace(area Area, fromName, toName string, surface string) (string, error) {
	cmd := fmt.Sprintf("/tiles-replace %s %s %s", area, fromName, toName)
	if surface != "" {
		cmd += " " + surface
	}
	return c.conn.Rcon(cmd)
}

// Checkerboard fills the area with an alternating pattern of two tile types.
// Corresponds to: /tiles-checker x1,y1,x2,y2 tileA tileB [surface]
func (c *Client) Checkerboard(area Area, tileA, tileB string, surface string) (string, error) {
	cmd := fmt.Sprintf("/tiles-checker %s %s %s", area, tileA, tileB)
	if surface != "" {
		cmd += " " + surface
	}
	return c.conn.Rcon(cmd)
}

// Parse converts the compact tiles-read wire format into a slice of Tile.
// The wire format is: name:x:y,name:x:y,...
// An empty string returns nil, nil.
func Parse(s string) ([]Tile, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	entries := strings.Split(s, ",")
	tiles := make([]Tile, 0, len(entries))
	for _, entry := range entries {
		parts := strings.SplitN(entry, ":", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid tile entry %q: expected name:x:y", entry)
		}
		x, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid x in %q: %w", entry, err)
		}
		y, err := strconv.Atoi(parts[2])
		if err != nil {
			return nil, fmt.Errorf("invalid y in %q: %w", entry, err)
		}
		tiles = append(tiles, Tile{Name: parts[0], X: x, Y: y})
	}
	return tiles, nil
}
