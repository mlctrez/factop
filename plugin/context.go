package plugin

import (
	"log/slog"
	"sync"

	"github.com/mlctrez/factop/client"
	"github.com/mlctrez/factop/client/entity"
	"github.com/mlctrez/factop/client/game"
	"github.com/mlctrez/factop/client/player"
	"github.com/mlctrez/factop/client/playerattr"
	"github.com/mlctrez/factop/client/resource"
	"github.com/mlctrez/factop/client/surface"
	"github.com/mlctrez/factop/client/tile"
	"github.com/nats-io/nats.go"
)

// contextImpl implements Context. It wraps client.Conn, NATS conn,
// data dir, logger, done channel, and lazily-initialized typed clients.
type contextImpl struct {
	conn       *client.Conn
	nc         *nats.Conn
	dataDir    string
	logger     *slog.Logger
	done       chan struct{}
	pluginName string
	subs       []*nats.Subscription

	clientsOnce sync.Once
	clients     *clientsImpl
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

// Clients returns lazily-initialized typed client accessors.
func (c *contextImpl) Clients() Clients {
	c.clientsOnce.Do(func() {
		c.clients = &clientsImpl{conn: c.conn}
	})
	return c.clients
}

// clientsImpl provides lazily-initialized typed client accessors.
// Each client is constructed on first access and cached for subsequent calls.
type clientsImpl struct {
	conn *client.Conn

	gameOnce   sync.Once
	gameClient *game.Client

	entityOnce   sync.Once
	entityClient *entity.Client

	tileOnce   sync.Once
	tileClient *tile.Client

	surfaceOnce   sync.Once
	surfaceClient *surface.Client

	resourceOnce   sync.Once
	resourceClient *resource.Client

	playerOnce   sync.Once
	playerClient *player.Client

	playerAttrOnce   sync.Once
	playerAttrClient *playerattr.Client
}

func (c *clientsImpl) Game() *game.Client {
	c.gameOnce.Do(func() { c.gameClient = game.New(c.conn) })
	return c.gameClient
}

func (c *clientsImpl) Entity() *entity.Client {
	c.entityOnce.Do(func() { c.entityClient = entity.New(c.conn) })
	return c.entityClient
}

func (c *clientsImpl) Tile() *tile.Client {
	c.tileOnce.Do(func() { c.tileClient = tile.New(c.conn) })
	return c.tileClient
}

func (c *clientsImpl) Surface() *surface.Client {
	c.surfaceOnce.Do(func() { c.surfaceClient = surface.New(c.conn) })
	return c.surfaceClient
}

func (c *clientsImpl) Resource() *resource.Client {
	c.resourceOnce.Do(func() { c.resourceClient = resource.New(c.conn) })
	return c.resourceClient
}

func (c *clientsImpl) Player() *player.Client {
	c.playerOnce.Do(func() { c.playerClient = player.New(c.conn) })
	return c.playerClient
}

func (c *clientsImpl) PlayerAttr() *playerattr.Client {
	c.playerAttrOnce.Do(func() { c.playerAttrClient = playerattr.New(c.conn) })
	return c.playerAttrClient
}
