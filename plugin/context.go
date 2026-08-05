package plugin

import (
	"context"
	"log/slog"
	"sync"

	"github.com/mlctrez/factop/client"
	"github.com/nats-io/nats.go"
)

// contextImpl implements Context. It wraps client.Conn, NATS conn,
// data dir, logger, done channel, and plugin name.
type contextImpl struct {
	conn       *client.Conn
	nc         *nats.Conn
	dataDir    string
	logger     *slog.Logger
	done       chan struct{}
	pluginName string
	subs       []*nats.Subscription

	eventsOnce sync.Once
	events     *EventRouter
	goCtx      context.Context
	cancel     context.CancelFunc
}

// Rcon delegates to the underlying client.Conn.
func (c *contextImpl) Rcon(command string) (string, error) {
	return c.conn.Rcon(command)
}

// Subscribe registers a NATS subscription and tracks it for teardown.
func (c *contextImpl) Subscribe(subject string, handler func([]byte)) error {
	sub, err := c.nc.Subscribe(subject, func(msg *nats.Msg) {
		handler(msg.Data)
	})
	if err != nil {
		return err
	}
	c.subs = append(c.subs, sub)
	return nil
}

// Publish sends data on a NATS subject.
func (c *contextImpl) Publish(subject string, data []byte) error {
	return c.nc.Publish(subject, data)
}

// DataDir returns the plugin's persistent data directory path.
func (c *contextImpl) DataDir() string {
	return c.dataDir
}

// Logger returns the structured logger for this plugin.
func (c *contextImpl) Logger() *slog.Logger {
	return c.logger
}

// Done returns a channel that closes when the SDK initiates shutdown.
func (c *contextImpl) Done() <-chan struct{} {
	return c.done
}

// Events returns the event router instance, lazily initialized on first call.
func (c *contextImpl) Events() *EventRouter {
	c.eventsOnce.Do(func() {
		c.events = newEventRouter(c.nc, c.logger)
	})
	return c.events
}

// Ctx returns a cancellable context for the plugin lifecycle.
func (c *contextImpl) Ctx() context.Context {
	return c.goCtx
}
