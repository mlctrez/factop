// Package entity provides a typed Go client for the entity manipulation
// commands registered by softmod/factop/entity.lua.
package entity

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mlctrez/factop/client"
)

// Entity represents a found entity with its position and unit number.
type Entity struct {
	Name       string  `json:"name"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	UnitNumber uint64  `json:"unit_number"`
}

// Area defines a rectangular search region.
type Area struct {
	X1 float64
	Y1 float64
	X2 float64
	Y2 float64
}

func (a Area) String() string {
	return fmt.Sprintf("%g,%g,%g,%g", a.X1, a.Y1, a.X2, a.Y2)
}

// Position is a map coordinate.
type Position struct {
	X float64
	Y float64
}

func (p Position) String() string {
	return fmt.Sprintf("%g,%g", p.X, p.Y)
}

// FindOptions controls filtering for Find and Count operations.
// Use "_" or empty string to skip a filter field.
type FindOptions struct {
	Name    string
	Type    string
	Force   string
	Limit   int
	Surface string
}

// Client provides typed methods for each entity-* RCON command.
type Client struct {
	conn *client.Conn
}

// New creates an entities Client using the given connection.
func New(conn *client.Conn) *Client {
	return &Client{conn: conn}
}

// Create places a single entity at the given position.
// Direction uses Factorio direction names: "north", "south", "east", "west", etc.
// Pass empty strings for force/direction/surface to use defaults.
func (c *Client) Create(pos Position, name, force, direction, surface string) (string, error) {
	cmd := fmt.Sprintf("/entity-create %s %s", pos, name)
	if force != "" {
		cmd += " " + force
	}
	if direction != "" {
		if force == "" {
			cmd += " player"
		}
		cmd += " " + direction
	}
	if surface != "" {
		// pad missing optional args
		if force == "" {
			cmd += " player"
		}
		if direction == "" {
			cmd += " _"
		}
		cmd += " " + surface
	}
	return c.conn.Rcon(cmd)
}

// filterArg returns the value or "_" as a skip placeholder.
func filterArg(s string) string {
	if s == "" {
		return "_"
	}
	return s
}

// Find returns entities matching the filter in the given area.
func (c *Client) Find(area Area, opts FindOptions) ([]Entity, error) {
	cmd := fmt.Sprintf("/entity-find %s %s %s %s",
		area, filterArg(opts.Name), filterArg(opts.Type), filterArg(opts.Force))
	if opts.Limit > 0 {
		cmd += " " + strconv.Itoa(opts.Limit)
	} else {
		cmd += " _"
	}
	if opts.Surface != "" {
		cmd += " " + opts.Surface
	}
	raw, err := c.conn.Rcon(cmd)
	if err != nil {
		return nil, err
	}
	return Parse(raw)
}

// Count returns the number of entities matching the filter in the area.
func (c *Client) Count(area Area, opts FindOptions) (int, error) {
	cmd := fmt.Sprintf("/entity-count %s %s %s %s",
		area, filterArg(opts.Name), filterArg(opts.Type), filterArg(opts.Force))
	if opts.Surface != "" {
		cmd += " " + opts.Surface
	}
	raw, err := c.conn.Rcon(cmd)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(raw))
}

// Destroy removes entities matching the filter in the area.
func (c *Client) Destroy(area Area, opts FindOptions) (string, error) {
	cmd := fmt.Sprintf("/entity-destroy %s %s %s %s",
		area, filterArg(opts.Name), filterArg(opts.Type), filterArg(opts.Force))
	if opts.Limit > 0 {
		cmd += " " + strconv.Itoa(opts.Limit)
	} else {
		cmd += " _"
	}
	if opts.Surface != "" {
		cmd += " " + opts.Surface
	}
	return c.conn.Rcon(cmd)
}

// Parse converts the compact entity-find wire format into a slice of Entity.
// Wire format: name:x:y:unit_number,name:x:y:unit_number,...
func Parse(s string) ([]Entity, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	entries := strings.Split(s, ",")
	result := make([]Entity, 0, len(entries))
	for _, entry := range entries {
		parts := strings.SplitN(entry, ":", 4)
		if len(parts) != 4 {
			return nil, fmt.Errorf("invalid entity entry %q: expected name:x:y:unit_number", entry)
		}
		x, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid x in %q: %w", entry, err)
		}
		y, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid y in %q: %w", entry, err)
		}
		un, err := strconv.ParseUint(parts[3], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid unit_number in %q: %w", entry, err)
		}
		result = append(result, Entity{Name: parts[0], X: x, Y: y, UnitNumber: un})
	}
	return result, nil
}
