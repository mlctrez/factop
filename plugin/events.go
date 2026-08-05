package plugin

import (
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// EventRouter dispatches NATS messages from udp.incoming.<tag> subjects
// to registered typed handler functions.
type EventRouter struct {
	nc     *nats.Conn
	logger *slog.Logger

	mu sync.Mutex
	// tags maps event tag → subscription state
	tags map[string]*tagState
}

// tagState holds the subscription and handlers for a single event tag.
type tagState struct {
	sub   *nats.Subscription
	typed []typedHandler
	raw   []func([]byte)
}

// typedHandler wraps a parse function and a set of typed callbacks.
type typedHandler struct {
	parse    func([]byte) (any, error)
	handlers []func(any)
}

func newEventRouter(nc *nats.Conn, logger *slog.Logger) *EventRouter {
	return &EventRouter{
		nc:     nc,
		logger: logger,
		tags:   make(map[string]*tagState),
	}
}

// subscribe ensures a NATS subscription exists for the given tag.
// Must be called with r.mu held.
func (r *EventRouter) subscribe(tag string) error {
	ts := r.tags[tag]
	if ts == nil {
		ts = &tagState{}
		r.tags[tag] = ts
	}
	if ts.sub != nil {
		return nil // already subscribed
	}
	subject := "udp.incoming." + tag
	sub, err := r.nc.Subscribe(subject, func(msg *nats.Msg) {
		r.dispatch(tag, msg.Data)
	})
	if err != nil {
		return err
	}
	ts.sub = sub
	return nil
}

func (r *EventRouter) dispatch(tag string, data []byte) {
	r.mu.Lock()
	ts := r.tags[tag]
	if ts == nil {
		r.mu.Unlock()
		return
	}
	// Snapshot handlers under lock to avoid holding lock during callbacks.
	typed := ts.typed
	raw := make([]func([]byte), len(ts.raw))
	copy(raw, ts.raw)
	r.mu.Unlock()

	// Invoke typed handlers first.
	for _, th := range typed {
		val, err := th.parse(data)
		if err != nil {
			r.logger.Error("event parse error",
				"tag", tag,
				"error", err,
				"msg", truncate(data, 256),
			)
			return // skip all handlers for this message on parse failure
		}
		for _, fn := range th.handlers {
			r.safeCall(tag, fn, val)
		}
	}

	// Then raw handlers.
	for _, fn := range raw {
		r.safeCallRaw(tag, fn, data)
	}
}

func (r *EventRouter) safeCall(tag string, fn func(any), val any) {
	defer func() {
		if rv := recover(); rv != nil {
			r.logger.Error("handler panic", "tag", tag, "panic", rv)
		}
	}()
	fn(val)
}

func (r *EventRouter) safeCallRaw(tag string, fn func([]byte), data []byte) {
	defer func() {
		if rv := recover(); rv != nil {
			r.logger.Error("raw handler panic", "tag", tag, "panic", rv)
		}
	}()
	fn(data)
}

// truncate returns at most max bytes of data as a string.
func truncate(data []byte, max int) string {
	if len(data) <= max {
		return string(data)
	}
	return string(data[:max])
}

// Drain unsubscribes all active subscriptions with a timeout.
func (r *EventRouter) Drain(timeout time.Duration) {
	r.mu.Lock()
	subs := make([]*nats.Subscription, 0, len(r.tags))
	for _, ts := range r.tags {
		if ts.sub != nil {
			subs = append(subs, ts.sub)
		}
	}
	r.mu.Unlock()

	done := make(chan struct{})
	go func() {
		for _, sub := range subs {
			_ = sub.Drain()
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(timeout):
		r.logger.Warn("event router drain timed out", "timeout", timeout)
	}
}

// OnRaw registers a handler for any event tag. The handler receives
// the full message bytes without parsing.
func (r *EventRouter) OnRaw(tag string, fn func([]byte)) error {
	if tag == "" {
		return errors.New("event tag must not be empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.subscribe(tag); err != nil {
		return err
	}
	r.tags[tag].raw = append(r.tags[tag].raw, fn)
	return nil
}
