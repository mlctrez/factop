// Package playerattr provides a typed Go client for the per-player
// key-value attribute commands registered by softmod/factop/playerattr.lua.
package playerattr

import (
	"fmt"
	"strings"

	"github.com/mlctrez/factop/client"
)

// Client provides typed methods for each playerattr-* RCON command.
type Client struct {
	conn *client.Conn
}

// New creates a playerattr Client using the given connection.
func New(conn *client.Conn) *Client {
	return &Client{conn: conn}
}

// Set sets a player attribute key to the given value.
func (c *Client) Set(player, key, value string) error {
	_, err := c.conn.Rcon(fmt.Sprintf("/playerattr-set %s %s %s", player, key, value))
	return err
}

// Get returns the value of a player attribute key, or empty string if unset.
func (c *Client) Get(player, key string) (string, error) {
	return c.conn.Rcon(fmt.Sprintf("/playerattr-get %s %s", player, key))
}

// Del removes a player attribute key.
func (c *Client) Del(player, key string) error {
	_, err := c.conn.Rcon(fmt.Sprintf("/playerattr-del %s %s", player, key))
	return err
}

// List returns all attributes for a player as a map.
func (c *Client) List(player string) (map[string]string, error) {
	raw, err := c.conn.Rcon(fmt.Sprintf("/playerattr-list %s", player))
	if err != nil {
		return nil, err
	}
	return ParseList(raw)
}

// Keys returns all attribute keys for a player.
func (c *Client) Keys(player string) ([]string, error) {
	raw, err := c.conn.Rcon(fmt.Sprintf("/playerattr-keys %s", player))
	if err != nil {
		return nil, err
	}
	return ParseKeys(raw)
}

// ParseList parses the playerattr-list wire format: key:value,key:value,...
func ParseList(s string) (map[string]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return map[string]string{}, nil
	}
	result := make(map[string]string)
	entries := strings.Split(s, ",")
	for _, entry := range entries {
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid attribute entry %q: expected key:value", entry)
		}
		result[parts[0]] = parts[1]
	}
	return result, nil
}

// ParseKeys parses the playerattr-keys wire format: key,key,...
func ParseKeys(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return []string{}, nil
	}
	return strings.Split(s, ","), nil
}
