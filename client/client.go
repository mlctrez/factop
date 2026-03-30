// Package client provides a typed Go client for interacting with a factop
// server over NATS. It handles the RCON-over-NATS connection and provides
// the base for command-specific sub-packages.
//
// Convention: each softmod/factop/<name>.lua that registers custom commands
// gets a corresponding client/<name>/ package. The Lua file defines the
// server-side commands; the Go package provides the typed client methods.
//
//	softmod/factop/tiles.lua  →  client/tiles/  (TilesClient)
//	softmod/factop/foo.lua    →  client/foo/    (FooClient)
//
// All sub-packages accept a *client.Conn and use its Rcon method to send
// commands to the factop server.
package client

import (
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	DefaultTimeout = 10 * time.Second
	RconSubject    = "factop.rcon"
)

// Conn wraps a NATS connection for communicating with a factop server.
type Conn struct {
	nc      *nats.Conn
	timeout time.Duration
}

// Option configures a Conn.
type Option func(*Conn)

// WithTimeout sets the RCON request timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Conn) { c.timeout = d }
}

// Dial connects to the NATS server at the given URL and returns a Conn.
func Dial(natsURL string, opts ...Option) (*Conn, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", natsURL, err)
	}
	c := &Conn{nc: nc, timeout: DefaultTimeout}
	for _, o := range opts {
		o(c)
	}
	return c, nil
}

// Close closes the underlying NATS connection.
func (c *Conn) Close() {
	c.nc.Close()
}

// Rcon sends a raw RCON command string to the factop server and returns
// the response. This is the low-level method used by all sub-packages.
func (c *Conn) Rcon(command string) (string, error) {
	msg, err := c.nc.Request(RconSubject, []byte(command), c.timeout)
	if err != nil {
		if errors.Is(err, nats.ErrTimeout) {
			return "", fmt.Errorf("rcon timeout for %q", command)
		}
		return "", fmt.Errorf("rcon request: %w", err)
	}
	if msg.Header != nil {
		if errVal := msg.Header.Get("error"); errVal != "" {
			return "", errors.New(errVal)
		}
	}
	return string(msg.Data), nil
}
