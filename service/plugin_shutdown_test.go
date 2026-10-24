package service

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"syscall"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// logBuffer is a goroutine-safe io.Writer for capturing log output.
type logBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *logBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *logBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// captureLogs points pm at a fresh cancellable context and a captured logger.
func captureLogs(t *testing.T, pm *PluginManager) (*logBuffer, context.CancelFunc) {
	t.Helper()
	logs := &logBuffer{}
	pm.Logger = *slog.New(slog.NewTextHandler(logs, nil))
	ctx, cancel := context.WithCancel(context.Background())
	pm.Context = ctx
	t.Cleanup(cancel)
	return logs, cancel
}

func setExitGrace(t *testing.T, d time.Duration) {
	t.Helper()
	old := unexpectedExitGrace
	unexpectedExitGrace = d
	t.Cleanup(func() { unexpectedExitGrace = old })
}

func pluginPids(pm *PluginManager, names ...string) map[string]int {
	pids := map[string]int{}
	pm.mu.Lock()
	defer pm.mu.Unlock()
	for _, n := range names {
		inst := pm.instances[n]
		inst.mu.Lock()
		pids[n] = inst.pid
		inst.mu.Unlock()
	}
	return pids
}

// TestPluginsSignalledBySystemdDuringShutdown reproduces a factop service stop
// under KillMode=control-group: systemd SIGTERMs the plugin processes at the
// same moment as factop, so they exit before the service context is cancelled
// and before PluginManager.Shutdown marks them Stopping. Those exits must not
// be reported as crashes or trigger restarts.
func TestPluginsSignalledBySystemdDuringShutdown(t *testing.T) {
	setExitGrace(t, 500*time.Millisecond)
	fac := newTestFactorio()
	startFakeFactorio(t, fac, StateRunning)
	pm := newTestPluginManager(t, fac, testPlugins...)
	logs, cancel := captureLogs(t, pm)

	pm.autoStartPlugins()
	assertPluginStates(t, pm, PluginStarting, testPlugins...)
	pids := pluginPids(pm, testPlugins...)

	// systemd: SIGTERM to every process in the cgroup.
	for _, n := range testPlugins {
		require.NoError(t, syscall.Kill(pids[n], syscall.SIGTERM))
	}
	// Plugins are reaped by processMonitor before factop reacts.
	waitPluginStates(t, pm, PluginErrored, testPlugins...)

	// factop's signal handler: Service.Stop cancels the context, then the
	// binder shuts components down.
	cancel()
	require.NoError(t, pm.Shutdown())

	time.Sleep(unexpectedExitGrace + 300*time.Millisecond)
	out := logs.String()
	assert.NotContains(t, out, "exited unexpectedly")
	assert.NotContains(t, out, "restart attempt")
	assert.Equal(t, len(testPlugins), strings.Count(out, "plugin exited during service shutdown"), out)
	assert.Contains(t, out, "exit=signal:terminated")
	assert.Equal(t, pids, pluginPids(pm, testPlugins...), "no plugin may be relaunched")
}

// TestPluginCrashStillRestarted verifies a genuine plugin crash is still
// reported and restarted, and that a pending restart is abandoned once the
// service starts shutting down.
func TestPluginCrashStillRestarted(t *testing.T) {
	setExitGrace(t, 50*time.Millisecond)
	fac := newTestFactorio()
	startFakeFactorio(t, fac, StateRunning)
	pm := newTestPluginManager(t, fac, "alpha", "bravo")
	logs, cancel := captureLogs(t, pm)

	pm.autoStartPlugins()
	pids := pluginPids(pm, "alpha", "bravo")

	// alpha crashes and is restarted after the first backoff.
	require.NoError(t, syscall.Kill(pids["alpha"], syscall.SIGKILL))
	require.Eventually(t, func() bool {
		return strings.Contains(logs.String(), `msg="plugin exited unexpectedly" name=alpha version=0.0.1 exit=signal:killed`)
	}, 2*time.Second, 10*time.Millisecond, logs.String())
	require.Eventually(t, func() bool {
		return pm.pluginState("alpha") == PluginStarting && pluginPids(pm, "alpha")["alpha"] != pids["alpha"]
	}, 5*time.Second, 20*time.Millisecond, "alpha was not restarted")

	// bravo crashes, then the service stops during the restart backoff.
	require.NoError(t, syscall.Kill(pids["bravo"], syscall.SIGKILL))
	require.Eventually(t, func() bool {
		return strings.Contains(logs.String(), `msg="plugin restart attempt" name=bravo`)
	}, 2*time.Second, 10*time.Millisecond, logs.String())
	cancel()
	require.Eventually(t, func() bool {
		return strings.Contains(logs.String(), `msg="service shutting down, abandoning plugin restart" name=bravo`)
	}, 2*time.Second, 10*time.Millisecond, logs.String())
	time.Sleep(2500 * time.Millisecond)
	assert.Equal(t, PluginErrored, pm.pluginState("bravo"), "bravo must not be restarted during shutdown")
}

func TestDescribeExit(t *testing.T) {
	assert.Equal(t, "code:0", describeExit(nil))
}

// TestSystemdUnitKillModeMixed verifies the unit factop installs uses
// KillMode=mixed and otherwise matches the servicego defaults.
func TestSystemdUnitKillModeMixed(t *testing.T) {
	sv := &Service{}
	cfg := sv.Config()
	script, ok := cfg.Option["SystemdScript"].(string)
	require.True(t, ok, "SystemdScript option must be set")

	stub := func(s string) string { return s }
	tmpl := template.Must(template.New("").Funcs(template.FuncMap{"cmd": stub, "cmdEscape": stub}).Parse(script))
	data := struct {
		Name, Description, Path, ChRoot, WorkingDirectory, UserName     string
		ReloadSignal, PIDFile, Restart, SuccessExitStatus, LogDirectory string
		Dependencies, Arguments                                         []string
		EnvVars                                                         map[string]string
		HasOutputFileSupport, LogOutput                                 bool
		LimitNOFILE                                                     int
	}{
		Name: cfg.Name, Description: cfg.Description, Path: "/opt/servicego/factop/factop",
		WorkingDirectory: cfg.WorkingDirectory, Dependencies: cfg.Dependencies,
		Restart: cfg.Option["Restart"].(string), SuccessExitStatus: cfg.Option["SuccessExitStatus"].(string),
		LimitNOFILE: -1,
	}
	var out bytes.Buffer
	require.NoError(t, tmpl.Execute(&out, data))
	unit := out.String()
	for _, want := range []string{
		"KillMode=mixed",
		"ExecStart=/opt/servicego/factop/factop",
		"Restart=on-success",
		"SuccessExitStatus=1 2 8 SIGKILL",
		"RestartSec=120",
		"Requires=network.target",
	} {
		assert.Contains(t, unit, want)
	}
}
