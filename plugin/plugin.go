// Package plugin provides the Plugin SDK that plugin authors import.
// A plugin implements the Plugin interface and calls Run(impl) from main().
package plugin

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mlctrez/factop/client"
	"github.com/nats-io/nats.go"
)

// Plugin is the interface that plugin authors implement.
type Plugin interface {
	Setup(ctx Context) error
	Run(ctx Context) error
}

// Context provides plugin access to factop infrastructure.
type Context interface {
	Rcon(command string) (string, error)
	Subscribe(subject string, handler func([]byte)) error
	Publish(subject string, data []byte) error
	DataDir() string
	Logger() *slog.Logger
	Done() <-chan struct{}
	Events() *EventRouter
	Ctx() context.Context
}

// healthResponse is the JSON payload for health check responses.
type healthResponse struct {
	Name          string  `json:"name"`
	Version       string  `json:"version"`
	UptimeSeconds float64 `json:"uptime_seconds"`
}

// Run is the entry point called from a plugin's main().
// It handles all infrastructure: flag parsing, NATS connection,
// health check subscription, signal handling, and teardown.
func Run(impl Plugin) {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	natsURL := fs.String("nats-url", "nats://localhost:4222", "NATS server URL")
	dataDir := fs.String("data-dir", "", "Plugin data directory")
	pluginName := fs.String("plugin-name", "", "Plugin name")
	version := fs.String("version", "0.0.0", "Plugin version")
	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "flag parse error: %v\n", err)
		os.Exit(1)
	}

	if *pluginName == "" {
		fmt.Fprintln(os.Stderr, "missing required flag: --plugin-name")
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil)).With("plugin", *pluginName)

	// Connect client.Conn for RCON.
	conn, err := client.Dial(*natsURL)
	if err != nil {
		logger.Error("failed to connect client", "error", err)
		os.Exit(1)
	}

	// Connect raw NATS for subscribe/publish.
	nc, err := nats.Connect(*natsURL)
	if err != nil {
		conn.Close()
		logger.Error("failed to connect NATS", "error", err)
		os.Exit(1)
	}

	startTime := time.Now()
	done := make(chan struct{})

	// Subscribe to health check.
	healthSub, err := nc.Subscribe("plugin."+*pluginName+".health", func(msg *nats.Msg) {
		resp := healthResponse{
			Name:          *pluginName,
			Version:       *version,
			UptimeSeconds: time.Since(startTime).Seconds(),
		}
		data, _ := json.Marshal(resp)
		_ = msg.Respond(data)
	})
	if err != nil {
		nc.Close()
		conn.Close()
		logger.Error("failed to subscribe to health check", "error", err)
		os.Exit(1)
	}

	goCtx, cancel := context.WithCancel(context.Background())

	ctx := &contextImpl{
		conn:       conn,
		nc:         nc,
		dataDir:    *dataDir,
		logger:     logger,
		done:       done,
		pluginName: *pluginName,
		goCtx:      goCtx,
		cancel:     cancel,
	}

	// Handle SIGINT/SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		close(done)
	}()

	// Cancel the context when done is closed.
	go func() {
		<-done
		cancel()
	}()

	// Setup phase.
	if err := impl.Setup(ctx); err != nil {
		logger.Error("setup failed", "error", err)
		_ = healthSub.Drain()
		nc.Close()
		conn.Close()
		os.Exit(1)
	}

	// Run phase — blocks until plugin work is complete or done is closed.
	runErr := impl.Run(ctx)

	// Teardown.
	_ = healthSub.Drain()
	for _, sub := range ctx.subs {
		_ = sub.Drain()
	}
	// Drain event router subscriptions if it was initialized.
	if ctx.events != nil {
		ctx.events.Drain(5 * time.Second)
	}
	nc.Close()
	conn.Close()

	if runErr != nil {
		logger.Error("run failed", "error", runErr)
		os.Exit(1)
	}
	os.Exit(0)
}
