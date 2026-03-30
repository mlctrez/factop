// Package surface provides a typed Go client for the surface property and
// chunk management commands registered by softmod/factop/surface.lua.
package surface

import (
	"fmt"

	"github.com/mlctrez/factop/client"
)

// Info holds the parsed surface properties from surface-info.
type Info struct {
	Name                 string  `json:"name"`
	Index                int     `json:"index"`
	AlwaysDay            bool    `json:"always_day"`
	Daytime              float64 `json:"daytime"`
	Darkness             float64 `json:"darkness"`
	FreezeDaytime        bool    `json:"freeze_daytime"`
	PeacefulMode         bool    `json:"peaceful_mode"`
	NoEnemiesMode        bool    `json:"no_enemies_mode"`
	WindSpeed            float64 `json:"wind_speed"`
	WindOrientation      float64 `json:"wind_orientation"`
	SolarPowerMultiplier float64 `json:"solar_power_multiplier"`
	MinBrightness        float64 `json:"min_brightness"`
	TicksPerDay          int     `json:"ticks_per_day"`
	ShowClouds           bool    `json:"show_clouds"`
	GenerateWithLabTiles bool    `json:"generate_with_lab_tiles"`
}

// SurfaceEntry is a name:index pair from surface-list.
type SurfaceEntry struct {
	Name  string `json:"name"`
	Index int    `json:"index"`
}

// Client provides typed methods for each surface-* RCON command.
type Client struct {
	conn *client.Conn
}

// New creates a surface Client using the given connection.
func New(conn *client.Conn) *Client {
	return &Client{conn: conn}
}

// List returns all surfaces.
func (c *Client) List() ([]SurfaceEntry, error) {
	raw, err := c.conn.Rcon("/surface-list")
	if err != nil {
		return nil, err
	}
	return ParseList(raw)
}

// GetInfo returns surface properties.
func (c *Client) GetInfo(surface string) (*Info, error) {
	cmd := "/surface-info"
	if surface != "" {
		cmd += " " + surface
	}
	raw, err := c.conn.Rcon(cmd)
	if err != nil {
		return nil, err
	}
	return ParseInfo(raw)
}

// Set sets a surface property. Property must be a writable boolean or numeric property name.
func (c *Client) Set(property, value string, surface string) (string, error) {
	cmd := fmt.Sprintf("/surface-set %s %s", property, value)
	if surface != "" {
		cmd += " " + surface
	}
	return c.conn.Rcon(cmd)
}

// Generate requests chunk generation at the given position with optional radius.
func (c *Client) Generate(x, y float64, radius int, surface string) (string, error) {
	cmd := fmt.Sprintf("/surface-generate %g,%g", x, y)
	if radius > 0 {
		cmd += fmt.Sprintf(" %d", radius)
	}
	if surface != "" {
		if radius <= 0 {
			cmd += " 0"
		}
		cmd += " " + surface
	}
	return c.conn.Rcon(cmd)
}

// DeleteChunk deletes a chunk at the given chunk coordinates.
func (c *Client) DeleteChunk(cx, cy int, surface string) (string, error) {
	cmd := fmt.Sprintf("/surface-delete-chunk %d,%d", cx, cy)
	if surface != "" {
		cmd += " " + surface
	}
	return c.conn.Rcon(cmd)
}

// ClearPollution clears all pollution on the surface.
func (c *Client) ClearPollution(surface string) (string, error) {
	cmd := "/surface-clear-pollution"
	if surface != "" {
		cmd += " " + surface
	}
	return c.conn.Rcon(cmd)
}
