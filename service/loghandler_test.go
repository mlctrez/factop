package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"testing/quick"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLogger implements kardianos/service.Logger and counts calls.
type mockLogger struct {
	callCount atomic.Int64
}

func (m *mockLogger) Error(v ...interface{}) error   { m.callCount.Add(1); return nil }
func (m *mockLogger) Warning(v ...interface{}) error  { m.callCount.Add(1); return nil }
func (m *mockLogger) Info(v ...interface{}) error      { m.callCount.Add(1); return nil }
func (m *mockLogger) Errorf(format string, a ...interface{}) error {
	m.callCount.Add(1)
	return nil
}
func (m *mockLogger) Warningf(format string, a ...interface{}) error {
	m.callCount.Add(1)
	return nil
}
func (m *mockLogger) Infof(format string, a ...interface{}) error {
	m.callCount.Add(1)
	return nil
}

// randomRecord generates a random slog.Record for property testing.
type randomRecord struct {
	Level   slog.Level
	Message string
	Time    time.Time
	Attrs   []slog.Attr
}

func (r randomRecord) Generate(rng *rand.Rand, size int) reflect.Value {
	levels := []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError}
	msgs := []string{"test", "hello world", "starting server", "connection lost", ""}
	attrKeys := []string{"key", "host", "port", "user", "action"}
	attrVals := []string{"val", "localhost", "8080", "admin", "restart"}

	numAttrs := rng.Intn(4)
	attrs := make([]slog.Attr, numAttrs)
	for i := range numAttrs {
		attrs[i] = slog.String(attrKeys[rng.Intn(len(attrKeys))], attrVals[rng.Intn(len(attrVals))])
	}

	return reflect.ValueOf(randomRecord{
		Level:   levels[rng.Intn(len(levels))],
		Message: msgs[rng.Intn(len(msgs))],
		Time:    time.Now().Add(time.Duration(rng.Int63n(1000)) * time.Second),
		Attrs:   attrs,
	})
}

func (r randomRecord) toSlogRecord() slog.Record {
	rec := slog.NewRecord(r.Time, r.Level, r.Message, 0)
	rec.AddAttrs(r.Attrs...)
	return rec
}

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

// TestLogHandlerProperty2_OSLoggerAlwaysCalled - Property 2
// **Validates: Requirements 2.1, 2.3**
//
// For any slog.Record and any NATS connection state (nil or non-nil),
// the LogHandler's Handle method should always write the record to the
// kardianos/service logger. The OS logger call count should equal the
// total Handle invocation count.
func TestLogHandlerProperty2_OSLoggerAlwaysCalled(t *testing.T) {
	nc, cleanup := startTestNATS(t)
	defer cleanup()

	f := func(rr randomRecord, withNats bool) bool {
		ml := &mockLogger{}
		h := &LogHandler{Logger: ml}

		if withNats {
			h.SetNatsConn(nc)
		}

		rec := rr.toSlogRecord()
		err := h.Handle(context.Background(), rec)
		if err != nil {
			return false
		}

		return ml.callCount.Load() == 1
	}

	require.NoError(t, quick.Check(f, &quick.Config{MaxCount: 200}))
}

// TestLogHandlerProperty2_BatchCounting verifies that across N sequential
// Handle calls, the OS logger is called exactly N times.
func TestLogHandlerProperty2_BatchCounting(t *testing.T) {
	nc, cleanup := startTestNATS(t)
	defer cleanup()

	ml := &mockLogger{}
	h := &LogHandler{Logger: ml}
	h.SetNatsConn(nc)

	const n = 50
	for i := range n {
		rec := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
		rec.AddAttrs(slog.Int("i", i))
		require.NoError(t, h.Handle(context.Background(), rec))
	}

	assert.Equal(t, int64(n), ml.callCount.Load())
}

// TestLogHandlerProperty3_NATSJSONContainsRequiredFields - Property 3
// **Validates: Requirements 2.4**
//
// For any slog.Record with arbitrary level, message, timestamp, and attributes,
// when a NATS connection is registered, the JSON payload published to factop.log
// should contain the keys time, level, msg, and attrs.
func TestLogHandlerProperty3_NATSJSONContainsRequiredFields(t *testing.T) {
	nc, cleanup := startTestNATS(t)
	defer cleanup()

	// Subscribe to factop.log to capture messages
	var mu sync.Mutex
	var captured [][]byte

	sub, err := nc.Subscribe("factop.log", func(msg *nats.Msg) {
		mu.Lock()
		defer mu.Unlock()
		captured = append(captured, msg.Data)
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	f := func(rr randomRecord) bool {
		ml := &mockLogger{}
		h := &LogHandler{Logger: ml}
		h.SetNatsConn(nc)

		rec := rr.toSlogRecord()
		if err := h.Handle(context.Background(), rec); err != nil {
			return false
		}

		// Flush to ensure message delivery
		nc.Flush()
		// Small sleep to allow subscription handler to process
		time.Sleep(10 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()

		if len(captured) == 0 {
			return false
		}

		last := captured[len(captured)-1]
		var payload map[string]any
		if err := json.Unmarshal(last, &payload); err != nil {
			return false
		}

		// Verify required keys exist
		if _, ok := payload["time"]; !ok {
			return false
		}
		if _, ok := payload["level"]; !ok {
			return false
		}
		if _, ok := payload["msg"]; !ok {
			return false
		}
		if _, ok := payload["attrs"]; !ok {
			return false
		}

		// Verify values correspond to the record
		if payload["level"] != rec.Level.String() {
			return false
		}
		if payload["msg"] != rec.Message {
			return false
		}

		return true
	}

	require.NoError(t, quick.Check(f, &quick.Config{MaxCount: 100}))
}

// TestLogHandlerProperty4_PublishFailureDoesNotDisrupt - Property 4
// **Validates: Requirements 2.5**
//
// For any slog.Record, when the NATS connection is nil (simulating the
// silent skip path for unavailable/failed NATS), Handle should still
// return nil and the OS service logger should still receive the record.
//
// Additionally tests with a closed NATS connection to trigger actual
// publish errors.
func TestLogHandlerProperty4_PublishFailureDoesNotDisrupt(t *testing.T) {
	t.Run("nil_conn", func(t *testing.T) {
		f := func(rr randomRecord) bool {
			ml := &mockLogger{}
			h := &LogHandler{Logger: ml}
			// No NATS connection set — simulates pre-registration state

			rec := rr.toSlogRecord()
			err := h.Handle(context.Background(), rec)
			if err != nil {
				return false
			}
			return ml.callCount.Load() == 1
		}

		require.NoError(t, quick.Check(f, &quick.Config{MaxCount: 200}))
	})

	t.Run("closed_conn", func(t *testing.T) {
		// Create a NATS connection and immediately close it to trigger publish errors
		nc, cleanup := startTestNATS(t)
		nc.Close() // close the connection so Publish will fail
		defer cleanup()

		f := func(rr randomRecord) bool {
			ml := &mockLogger{}
			h := &LogHandler{Logger: ml}
			h.SetNatsConn(nc) // set the closed connection

			rec := rr.toSlogRecord()
			err := h.Handle(context.Background(), rec)
			if err != nil {
				return false
			}
			return ml.callCount.Load() == 1
		}

		require.NoError(t, quick.Check(f, &quick.Config{MaxCount: 200}))
	})
}

// TestLogHandlerWithAttrsWithGroup verifies that WithAttrs and WithGroup
// return the receiver without panicking (no-op implementations).
func TestLogHandlerWithAttrsWithGroup(t *testing.T) {
	ml := &mockLogger{}
	h := &LogHandler{Logger: ml}

	t.Run("WithAttrs_returns_receiver", func(t *testing.T) {
		result := h.WithAttrs([]slog.Attr{slog.String("key", "val")})
		assert.Same(t, h, result, "WithAttrs should return the same handler")
	})

	t.Run("WithAttrs_nil_attrs", func(t *testing.T) {
		result := h.WithAttrs(nil)
		assert.Same(t, h, result, "WithAttrs(nil) should return the same handler")
	})

	t.Run("WithGroup_returns_receiver", func(t *testing.T) {
		result := h.WithGroup("testgroup")
		assert.Same(t, h, result, "WithGroup should return the same handler")
	})

	t.Run("WithGroup_empty_name", func(t *testing.T) {
		result := h.WithGroup("")
		assert.Same(t, h, result, "WithGroup('') should return the same handler")
	})
}
