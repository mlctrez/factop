package plugin

import (
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/mlctrez/factop/client/player"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startTestNATS starts an embedded NATS server and returns a connection + cleanup func.
func startTestNATS(t *testing.T) (*nats.Conn, func()) {
	t.Helper()
	opts := &server.Options{Host: "127.0.0.1", Port: -1, NoSigs: true}
	ns, err := server.NewServer(opts)
	require.NoError(t, err)
	go ns.Start()
	require.True(t, ns.ReadyForConnections(5*time.Second), "NATS server not ready")

	nc, err := nats.Connect(ns.ClientURL())
	require.NoError(t, err)

	return nc, func() {
		nc.Close()
		ns.Shutdown()
	}
}

func TestEventRouter_Dispatch(t *testing.T) {
	nc, cleanup := startTestNATS(t)
	defer cleanup()

	logger := slog.Default()
	router := newEventRouter(nc, logger)

	received := make(chan []byte, 1)
	err := router.OnRaw("test", func(data []byte) {
		received <- data
	})
	require.NoError(t, err)

	// Publish a message on the subject the router subscribes to.
	err = nc.Publish("udp.incoming.test", []byte("hello world"))
	require.NoError(t, err)
	require.NoError(t, nc.Flush())

	select {
	case msg := <-received:
		assert.Equal(t, []byte("hello world"), msg)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for handler invocation")
	}
}

func TestEventRouter_PanicRecovery(t *testing.T) {
	nc, cleanup := startTestNATS(t)
	defer cleanup()

	logger := slog.Default()
	router := newEventRouter(nc, logger)

	afterPanic := make(chan []byte, 1)

	// First handler panics.
	err := router.OnRaw("panic", func(data []byte) {
		panic("handler exploded")
	})
	require.NoError(t, err)

	// Second handler should still fire.
	err = router.OnRaw("panic", func(data []byte) {
		afterPanic <- data
	})
	require.NoError(t, err)

	err = nc.Publish("udp.incoming.panic", []byte("boom"))
	require.NoError(t, err)
	require.NoError(t, nc.Flush())

	select {
	case msg := <-afterPanic:
		assert.Equal(t, []byte("boom"), msg)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out — panic in first handler prevented second handler from firing")
	}
}

func TestEventRouter_MultipleHandlers(t *testing.T) {
	nc, cleanup := startTestNATS(t)
	defer cleanup()

	logger := slog.Default()
	router := newEventRouter(nc, logger)

	var mu sync.Mutex
	var order []int

	for i := range 3 {
		idx := i
		err := router.OnRaw("multi", func(data []byte) {
			mu.Lock()
			order = append(order, idx)
			mu.Unlock()
		})
		require.NoError(t, err)
	}

	err := nc.Publish("udp.incoming.multi", []byte("msg"))
	require.NoError(t, err)
	require.NoError(t, nc.Flush())

	// Wait for all handlers to fire.
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(order) == 3
	}, 2*time.Second, 10*time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []int{0, 1, 2}, order, "handlers should fire in registration order")
}

func TestEventRouter_ParseError(t *testing.T) {
	nc, cleanup := startTestNATS(t)
	defer cleanup()

	logger := slog.Default()
	router := newEventRouter(nc, logger)

	// Manually add a typed handler with a parser that always fails.
	router.mu.Lock()
	ts := &tagState{}
	router.tags["parsefail"] = ts
	sub, err := nc.Subscribe("udp.incoming.parsefail", func(msg *nats.Msg) {
		router.dispatch("parsefail", msg.Data)
	})
	require.NoError(t, err)
	ts.sub = sub

	typedCalled := make(chan struct{}, 1)
	ts.typed = []typedHandler{
		{
			parse: func(data []byte) (any, error) {
				return nil, errors.New("parse failed")
			},
			handlers: []func(any){
				func(v any) {
					typedCalled <- struct{}{}
				},
			},
		},
	}

	// Also register a raw handler — it should still fire even when parse fails.
	rawReceived := make(chan []byte, 1)
	ts.raw = []func([]byte){
		func(data []byte) {
			rawReceived <- data
		},
	}
	router.mu.Unlock()

	err = nc.Publish("udp.incoming.parsefail", []byte("bad data"))
	require.NoError(t, err)
	require.NoError(t, nc.Flush())

	// Raw handler should NOT fire on parse error per design:
	// "return // skip all handlers for this message on parse failure"
	// Wait a bit to confirm typed handler was NOT called.
	select {
	case <-typedCalled:
		t.Fatal("typed handler should not have been called on parse error")
	case <-time.After(200 * time.Millisecond):
		// Expected: typed handler was not invoked.
	}

	// Raw handler also should NOT fire because dispatch returns early on parse error.
	select {
	case <-rawReceived:
		t.Fatal("raw handler should not have been called on parse error (dispatch returns early)")
	case <-time.After(200 * time.Millisecond):
		// Expected: raw handler was not invoked.
	}
}

func TestEventRouter_Drain(t *testing.T) {
	nc, cleanup := startTestNATS(t)
	defer cleanup()

	logger := slog.Default()
	router := newEventRouter(nc, logger)

	received := make(chan []byte, 1)
	err := router.OnRaw("drain1", func(data []byte) {
		received <- data
	})
	require.NoError(t, err)
	err = router.OnRaw("drain2", func(data []byte) {})
	require.NoError(t, err)

	// Verify subscriptions are active before drain.
	router.mu.Lock()
	assert.NotNil(t, router.tags["drain1"].sub)
	assert.NotNil(t, router.tags["drain2"].sub)
	router.mu.Unlock()

	// Drain should complete without error.
	router.Drain(5 * time.Second)

	// After drain, publishing to the subject should not reach the handler
	// because the subscription has been drained.
	err = nc.Publish("udp.incoming.drain1", []byte("after drain"))
	require.NoError(t, err)
	require.NoError(t, nc.Flush())

	select {
	case <-received:
		t.Fatal("handler should not receive messages after drain")
	case <-time.After(200 * time.Millisecond):
		// Expected: no message delivered after drain.
	}
}

func TestEventRouter_OnRawEmptyTag(t *testing.T) {
	nc, cleanup := startTestNATS(t)
	defer cleanup()

	logger := slog.Default()
	router := newEventRouter(nc, logger)

	err := router.OnRaw("", func(data []byte) {})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestEventRouter_OnMove(t *testing.T) {
	nc, cleanup := startTestNATS(t)
	defer cleanup()

	logger := slog.Default()
	router := newEventRouter(nc, logger)

	received := make(chan player.Move, 1)
	err := router.OnMove(func(mv player.Move) {
		received <- mv
	})
	require.NoError(t, err)

	// Publish a valid move message to the subject the router subscribes to.
	msg := "[move] player1:1:45.5:-12.0:nauvis:1"
	err = nc.Publish("udp.incoming.move", []byte(msg))
	require.NoError(t, err)
	require.NoError(t, nc.Flush())

	select {
	case mv := <-received:
		assert.Equal(t, "player1", mv.Name)
		assert.Equal(t, 1, mv.Index)
		assert.Equal(t, 45.5, mv.X)
		assert.Equal(t, -12.0, mv.Y)
		assert.Equal(t, "nauvis", mv.SurfaceName)
		assert.Equal(t, 1, mv.SurfaceIndex)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for OnMove handler invocation")
	}

	// Test that a malformed message does NOT invoke the handler.
	malformed := "[move] bad:data"
	err = nc.Publish("udp.incoming.move", []byte(malformed))
	require.NoError(t, err)
	require.NoError(t, nc.Flush())

	select {
	case mv := <-received:
		t.Fatalf("handler should not have been called for malformed message, got: %+v", mv)
	case <-time.After(200 * time.Millisecond):
		// Expected: handler not invoked on parse error.
	}
}
