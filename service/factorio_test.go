package service

import (
	"io"
	"log/slog"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/quick"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestFactorio creates a Factorio struct with a discard logger for testing.
func newTestFactorio() *Factorio {
	return &Factorio{
		Logger: *slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

// allStates enumerates every defined ServerState value.
var allStates = []ServerState{StateStopped, StateStarting, StateRunning, StateStopping, StateError}

// statePair is a (from, to) pair used by testing/quick generators.
type statePair struct {
	From, To ServerState
}

func (statePair) Generate(r *rand.Rand, _ int) reflect.Value {
	return reflect.ValueOf(statePair{
		From: ServerState(r.Intn(len(allStates))),
		To:   ServerState(r.Intn(len(allStates))),
	})
}

// isValidTransition checks the exported transition table.
func isValidTransition(from, to ServerState) bool {
	if targets, ok := validTransitions[from]; ok {
		return targets[to]
	}
	return false
}

// TestFactorioProperty5_InvalidTransitionsRejected - Property 5
// **Validates: Requirements 3.2, 3.5, 3.9**
//
// For any (state, operation) pair not in the valid transition table,
// requesting that operation should return a non-nil error whose message
// contains both the current state name and the target state name.
func TestFactorioProperty5_InvalidTransitionsRejected(t *testing.T) {
	f := func(sp statePair) bool {
		if isValidTransition(sp.From, sp.To) {
			return true // skip valid pairs
		}

		fac := newTestFactorio()
		fac.mu.Lock()
		fac.state = sp.From
		err := fac.transition(sp.To)
		fac.mu.Unlock()

		if err == nil {
			return false
		}
		msg := err.Error()
		return strings.Contains(msg, sp.From.String()) && strings.Contains(msg, sp.To.String())
	}

	require.NoError(t, quick.Check(f, &quick.Config{MaxCount: 500}))
}

// TestFactorioProperty6_UnexpectedExitTransitionsToError - Property 6
// **Validates: Requirements 3.7**
//
// For any state in {Starting, Running}, when the monitor goroutine detects
// a process exit without a preceding stop request, the state machine should
// transition to Error.
func TestFactorioProperty6_UnexpectedExitToError(t *testing.T) {
	type activeState struct {
		State ServerState
	}
	gen := func(r *rand.Rand, _ int) reflect.Value {
		states := []ServerState{StateStarting, StateRunning}
		return reflect.ValueOf(activeState{State: states[r.Intn(2)]})
	}

	f := func(as activeState) bool {
		fac := newTestFactorio()
		fac.mu.Lock()
		fac.state = as.State
		// Simulate what processMonitor does on unexpected exit:
		// if state is Starting or Running, transition to Error.
		if fac.state == StateStarting || fac.state == StateRunning {
			if err := fac.transition(StateError); err != nil {
				fac.mu.Unlock()
				return false
			}
		}
		result := fac.state
		fac.mu.Unlock()
		return result == StateError
	}

	cfg := &quick.Config{
		MaxCount: 200,
		Values: func(values []reflect.Value, r *rand.Rand) {
			values[0] = gen(r, 0)
		},
	}
	require.NoError(t, quick.Check(f, cfg))
}

// --- Unit Tests ---

// TestFactorioValidTransitions verifies specific valid transitions succeed.
func TestFactorioValidTransitions(t *testing.T) {
	cases := []struct {
		name     string
		from, to ServerState
	}{
		{"Stopped→Starting", StateStopped, StateStarting},
		{"Starting→Running", StateStarting, StateRunning},
		{"Running→Stopping", StateRunning, StateStopping},
		{"Stopping→Stopped", StateStopping, StateStopped},
		{"Error→Starting", StateError, StateStarting},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fac := newTestFactorio()
			fac.mu.Lock()
			fac.state = tc.from
			err := fac.transition(tc.to)
			got := fac.state
			fac.mu.Unlock()

			assert.NoError(t, err)
			assert.Equal(t, tc.to, got)
		})
	}
}

// TestFactorioInvalidTransitions verifies specific invalid transitions return errors.
func TestFactorioInvalidTransitions(t *testing.T) {
	cases := []struct {
		name     string
		from, to ServerState
	}{
		{"Running→Starting", StateRunning, StateStarting},
		{"Stopped→Stopping", StateStopped, StateStopping},
		{"Stopped→Running", StateStopped, StateRunning},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fac := newTestFactorio()
			fac.mu.Lock()
			fac.state = tc.from
			err := fac.transition(tc.to)
			got := fac.state
			fac.mu.Unlock()

			assert.Error(t, err)
			assert.Equal(t, tc.from, got, "state should not change on invalid transition")
			assert.Contains(t, err.Error(), tc.from.String())
			assert.Contains(t, err.Error(), tc.to.String())
		})
	}
}

// TestFactorioRconMarkerDetection verifies that stdoutStateMonitor transitions
// from Starting to Running when it receives a message containing RconStartupMarker.
func TestFactorioRconMarkerDetection(t *testing.T) {
	fac := newTestFactorio()
	fac.mu.Lock()
	fac.state = StateStarting
	fac.mu.Unlock()

	// Simulate a NATS message containing the RCON startup marker
	msg := &nats.Msg{Data: []byte("0.000 " + RconStartupMarker + " (127.0.0.1:3000)")}
	fac.stdoutStateMonitor(msg)

	fac.mu.Lock()
	got := fac.state
	fac.mu.Unlock()

	assert.Equal(t, StateRunning, got, "state should transition to Running after RCON marker")
}

// TestFactorioRconMarkerIgnoredWhenNotStarting verifies that the RCON marker
// does not cause a transition when the server is not in Starting state.
func TestFactorioRconMarkerIgnoredWhenNotStarting(t *testing.T) {
	for _, state := range []ServerState{StateStopped, StateRunning, StateStopping, StateError} {
		t.Run(state.String(), func(t *testing.T) {
			fac := newTestFactorio()
			fac.mu.Lock()
			fac.state = state
			fac.mu.Unlock()

			msg := &nats.Msg{Data: []byte(RconStartupMarker)}
			fac.stdoutStateMonitor(msg)

			fac.mu.Lock()
			got := fac.state
			fac.mu.Unlock()

			assert.Equal(t, state, got, "state should not change when not in Starting state")
		})
	}
}
