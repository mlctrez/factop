// Package resources provides a typed Go client for the resource and pollution
// commands registered by softmod/factop/resources.lua.
package resources

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mlctrez/factop/client"
)

// ResourceCount is a resource name and its total count on a surface.
type ResourceCount struct {
	Name  string `json:"name"`
	Count uint64 `json:"count"`
}

// Resource is a resource entity with position and amount.
type Resource struct {
	Name   string  `json:"name"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Amount uint32  `json:"amount"`
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

// Client provides typed methods for each resources-* and pollution-* RCON command.
type Client struct {
	conn *client.Conn
}

// New creates a resources Client using the given connection.
func New(conn *client.Conn) *Client {
	return &Client{conn: conn}
}

// Count returns total resource counts for the surface.
func (c *Client) Count(surface string) ([]ResourceCount, error) {
	cmd := "/resources-count"
	if surface != "" {
		cmd += " " + surface
	}
	raw, err := c.conn.Rcon(cmd)
	if err != nil {
		return nil, err
	}
	return ParseCounts(raw)
}

// Find returns resource entities in the given area.
func (c *Client) Find(area Area, name string, limit int, surface string) ([]Resource, error) {
	nameArg := name
	if nameArg == "" {
		nameArg = "_"
	}
	limitArg := "_"
	if limit > 0 {
		limitArg = strconv.Itoa(limit)
	}
	cmd := fmt.Sprintf("/resources-find %s %s %s", area, nameArg, limitArg)
	if surface != "" {
		cmd += " " + surface
	}
	raw, err := c.conn.Rcon(cmd)
	if err != nil {
		return nil, err
	}
	return ParseResources(raw)
}

// SetAmount sets the resource amount at the given position.
func (c *Client) SetAmount(x, y float64, amount uint32, surface string) (string, error) {
	cmd := fmt.Sprintf("/resources-set %g,%g %d", x, y, amount)
	if surface != "" {
		cmd += " " + surface
	}
	return c.conn.Rcon(cmd)
}

// GetPollution returns the pollution value at the given position.
func (c *Client) GetPollution(x, y float64, surface string) (float64, error) {
	cmd := fmt.Sprintf("/pollution-get %g,%g", x, y)
	if surface != "" {
		cmd += " " + surface
	}
	raw, err := c.conn.Rcon(cmd)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(strings.TrimSpace(raw), 64)
}

// SetPollution sets the pollution value at the given position.
func (c *Client) SetPollution(x, y, amount float64, surface string) (string, error) {
	cmd := fmt.Sprintf("/pollution-set %g,%g %g", x, y, amount)
	if surface != "" {
		cmd += " " + surface
	}
	return c.conn.Rcon(cmd)
}

// TotalPollution returns the total pollution on the surface.
func (c *Client) TotalPollution(surface string) (float64, error) {
	cmd := "/pollution-total"
	if surface != "" {
		cmd += " " + surface
	}
	raw, err := c.conn.Rcon(cmd)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(strings.TrimSpace(raw), 64)
}

// ParseCounts parses the resources-count wire format: name:count,name:count,...
func ParseCounts(s string) ([]ResourceCount, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	entries := strings.Split(s, ",")
	result := make([]ResourceCount, 0, len(entries))
	for _, entry := range entries {
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid resource count entry %q", entry)
		}
		count, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid count in %q: %w", entry, err)
		}
		result = append(result, ResourceCount{Name: parts[0], Count: count})
	}
	return result, nil
}

// ParseResources parses the resources-find wire format: name:x:y:amount,...
func ParseResources(s string) ([]Resource, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	entries := strings.Split(s, ",")
	result := make([]Resource, 0, len(entries))
	for _, entry := range entries {
		parts := strings.SplitN(entry, ":", 4)
		if len(parts) != 4 {
			return nil, fmt.Errorf("invalid resource entry %q: expected name:x:y:amount", entry)
		}
		x, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid x in %q: %w", entry, err)
		}
		y, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid y in %q: %w", entry, err)
		}
		amount, err := strconv.ParseUint(parts[3], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid amount in %q: %w", entry, err)
		}
		result = append(result, Resource{Name: parts[0], X: x, Y: y, Amount: uint32(amount)})
	}
	return result, nil
}
