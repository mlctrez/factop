package service

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakePluginBinary writes a shell script that stands in for a plugin binary.
// It ignores its arguments and sleeps until signalled.
func fakePluginBinary(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fakeplugin")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\nexec sleep 300\n"), 0755))
	return path
}

// startFakeFactorio launches a stand-in process for the Factorio server and
// puts the state machine into the given state with processMonitor running.
func startFakeFactorio(t *testing.T, f *Factorio, state ServerState) {
	t.Helper()
	cmd := exec.Command("sleep", "300")
	require.NoError(t, cmd.Start())
	f.mu.Lock()
	f.cmd = cmd
	f.done = make(chan struct{})
	f.state = state
	f.mu.Unlock()
	go f.processMonitor()
	t.Cleanup(func() { _ = cmd.Process.Kill() })
}

// newTestPluginManager builds a PluginManager wired to fac (including the
// Factorio stop hook) with the named enabled plugins registered.
func newTestPluginManager(t *testing.T, fac *Factorio, names ...string) *PluginManager {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	bin := fakePluginBinary(t)
	reg := &PluginRegistry{}
	for _, n := range names {
		reg.Plugins = append(reg.Plugins, PluginEntry{Name: n, Version: "0.0.1", BinaryPath: bin, Enabled: true})
	}
	pm := &PluginManager{
		Logger:    *slog.New(slog.NewTextHandler(io.Discard, nil)),
		Context:   ctx,
		Factorio:  fac,
		registry:  reg,
		instances: make(map[string]*pluginInstance),
		chunks:    make(map[string]*deployBuffer),
	}
	fac.OnStop(pm.stopAllPlugins)
	t.Cleanup(func() {
		pm.stopAllPlugins()
		cancel()
	})
	return pm
}

func assertPluginStates(t *testing.T, pm *PluginManager, want PluginState, names ...string) {
	t.Helper()
	for _, n := range names {
		assert.Equal(t, want.String(), pm.pluginState(n).String(), "plugin %s", n)
	}
}

func waitPluginStates(t *testing.T, pm *PluginManager, want PluginState, names ...string) {
	t.Helper()
	require.Eventually(t, func() bool {
		for _, n := range names {
			if pm.pluginState(n) != want {
				return false
			}
		}
		return true
	}, 10*time.Second, 20*time.Millisecond, "plugins did not reach %s", want)
}

var testPlugins = []string{"alpha", "bravo", "charlie"}

// TestPluginsStoppedBeforeFactorioShutdown verifies that a graceful Factorio
// stop shuts plugins down before the server process is signalled, and that
// they stay stopped afterwards.
func TestPluginsStoppedBeforeFactorioShutdown(t *testing.T) {
	fac := newTestFactorio()
	startFakeFactorio(t, fac, StateRunning)
	pm := newTestPluginManager(t, fac, testPlugins...)

	pm.autoStartPlugins()
	assertPluginStates(t, pm, PluginStarting, testPlugins...)

	serverAliveDuringHook := false
	fac.mu.Lock()
	done := fac.done
	fac.mu.Unlock()
	fac.OnStop(func() {
		// Registered after the plugin manager hook, so plugins are already stopped.
		select {
		case <-done:
		default:
			serverAliveDuringHook = true
		}
		for _, n := range testPlugins {
			assert.Equal(t, PluginStopped, pm.pluginState(n), "plugin %s stopped before server signalled", n)
		}
	})

	require.NoError(t, fac.Shutdown())
	assert.True(t, serverAliveDuringHook, "stop hooks must run before the server process exits")
	assert.Equal(t, StateStopped.String(), fac.Status())
	assertPluginStates(t, pm, PluginStopped, testPlugins...)

	// No RCON marker arrives while stopped, so plugins must remain stopped.
	time.Sleep(200 * time.Millisecond)
	assertPluginStates(t, pm, PluginStopped, testPlugins...)
}

// TestPluginsRestartedWithFactorio verifies the restart cycle: plugins stop
// when Factorio goes down and are auto-started again when it reports RCON up,
// without any "invalid plugin state transition" errors.
func TestPluginsRestartedWithFactorio(t *testing.T) {
	fac := newTestFactorio()
	startFakeFactorio(t, fac, StateRunning)
	pm := newTestPluginManager(t, fac, testPlugins...)

	pm.autoStartPlugins()
	assertPluginStates(t, pm, PluginStarting, testPlugins...)
	firstPids := map[string]int{}
	for _, n := range testPlugins {
		firstPids[n] = pm.instances[n].pid
	}

	// Equivalent of the first half of Factorio.Restart.
	require.NoError(t, fac.Shutdown())
	assertPluginStates(t, pm, PluginStopped, testPlugins...)

	// Second half: new process comes up and prints the RCON marker.
	startFakeFactorio(t, fac, StateStarting)
	marker := &nats.Msg{Data: []byte("0.000 " + RconStartupMarker + " (127.0.0.1:3000)")}
	fac.stdoutStateMonitor(marker)
	pm.stdoutMonitor(marker)

	waitPluginStates(t, pm, PluginStarting, testPlugins...)
	for _, n := range testPlugins {
		pm.mu.Lock()
		pid := pm.instances[n].pid
		pm.mu.Unlock()
		assert.NotEqual(t, firstPids[n], pid, "plugin %s should be a new process", n)
	}
}

// TestPluginsStoppedOnFactorioCrash verifies that an unexpected Factorio exit
// stops plugins and does not trigger plugin restarts.
func TestPluginsStoppedOnFactorioCrash(t *testing.T) {
	fac := newTestFactorio()
	startFakeFactorio(t, fac, StateRunning)
	pm := newTestPluginManager(t, fac, testPlugins...)

	pm.autoStartPlugins()
	assertPluginStates(t, pm, PluginStarting, testPlugins...)

	fac.mu.Lock()
	proc := fac.cmd.Process
	fac.mu.Unlock()
	require.NoError(t, proc.Kill())

	waitPluginStates(t, pm, PluginStopped, testPlugins...)
	assert.Equal(t, StateError.String(), fac.Status())
}

// TestAutoStartSkipsActivePlugins verifies auto-start tolerates plugins that
// are already starting or running instead of failing the state transition.
func TestAutoStartSkipsActivePlugins(t *testing.T) {
	fac := newTestFactorio()
	fac.state = StateRunning
	pm := newTestPluginManager(t, fac, testPlugins...)

	pm.autoStartPlugins()
	assertPluginStates(t, pm, PluginStarting, testPlugins...)

	// Promote one plugin to Running as a successful health check would.
	inst := pm.instances["bravo"]
	inst.mu.Lock()
	require.NoError(t, inst.transition(PluginRunning))
	inst.mu.Unlock()

	pids := map[string]int{}
	for _, n := range testPlugins {
		pids[n] = pm.instances[n].pid
	}

	pm.autoStartPlugins()

	assert.Equal(t, PluginStarting, pm.pluginState("alpha"))
	assert.Equal(t, PluginRunning, pm.pluginState("bravo"))
	assert.Equal(t, PluginStarting, pm.pluginState("charlie"))
	for _, n := range testPlugins {
		assert.Equal(t, pids[n], pm.instances[n].pid, "plugin %s must not be relaunched", n)
	}
}

// TestFactorioShutdownWithoutHooks verifies Shutdown works with no hooks registered.
func TestFactorioShutdownWithoutHooks(t *testing.T) {
	fac := newTestFactorio()
	startFakeFactorio(t, fac, StateRunning)
	require.NoError(t, fac.Shutdown())
	assert.Equal(t, StateStopped.String(), fac.Status())
}
