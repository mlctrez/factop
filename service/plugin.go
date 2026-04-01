package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/mlctrez/bind"
	"github.com/nats-io/nats.go"
)

const (
	PluginRegistryPath = "/opt/factorio/plugins/registry.json"
	PluginBaseDir      = "/opt/factorio/plugins"
)

// PluginState represents the lifecycle state of a managed plugin process.
type PluginState int

const (
	PluginStopped  PluginState = iota // No process running
	PluginStarting                    // Process launched, waiting for first health check
	PluginRunning                     // Health check OK, plugin operational
	PluginStopping                    // Interrupt sent, waiting for process exit
	PluginErrored                     // Unexpected exit or health check failure
)

func (s PluginState) String() string {
	switch s {
	case PluginStopped:
		return "stopped"
	case PluginStarting:
		return "starting"
	case PluginRunning:
		return "running"
	case PluginStopping:
		return "stopping"
	case PluginErrored:
		return "error"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}

// pluginValidTransitions defines the allowed state transitions for a plugin.
var pluginValidTransitions = map[PluginState]map[PluginState]bool{
	PluginStopped:  {PluginStarting: true},
	PluginErrored:  {PluginStarting: true},
	PluginStarting: {PluginRunning: true, PluginErrored: true, PluginStopping: true},
	PluginRunning:  {PluginStopping: true, PluginErrored: true},
	PluginStopping: {PluginStopped: true},
}

// pluginInstance tracks the runtime state of a single plugin process.
type pluginInstance struct {
	mu           sync.Mutex
	entry        PluginEntry
	state        PluginState
	cmd          *exec.Cmd
	pid          int
	lastSeen     time.Time
	restartCount int
	done         chan struct{}
}

// transition validates and performs a state transition. Must be called with pi.mu held.
func (pi *pluginInstance) transition(to PluginState) error {
	from := pi.state
	if targets, ok := pluginValidTransitions[from]; ok && targets[to] {
		pi.state = to
		return nil
	}
	return fmt.Errorf("invalid plugin state transition from %s to %s", from.String(), to.String())
}

var _ bind.Startup = (*PluginManager)(nil)
var _ bind.Shutdown = (*PluginManager)(nil)

// PluginManager manages external plugin processes: registration, lifecycle,
// health checking, versioning, and deployment.
type PluginManager struct {
	slog.Logger
	Context  context.Context
	Nats     *Nats
	Factorio *Factorio
	Settings *Settings

	mu        sync.Mutex
	registry  *PluginRegistry
	instances map[string]*pluginInstance
	// chunks tracks in-progress chunked deploys keyed by "name/version".
	chunks map[string]*deployBuffer
}

// deployBuffer accumulates chunks for a single chunked deploy.
type deployBuffer struct {
	total    int
	received map[int][]byte
}

func (pm *PluginManager) Startup() error {
	reg, err := LoadRegistry(PluginRegistryPath)
	if err != nil {
		return fmt.Errorf("loading plugin registry: %w", err)
	}
	pm.registry = reg

	for _, w := range pm.registry.Validate() {
		pm.Warn(w)
	}

	pm.instances = make(map[string]*pluginInstance)
	pm.chunks = make(map[string]*deployBuffer)

	if err := pm.Nats.Subscribe("factop.plugin", pm.handleCommand); err != nil {
		return fmt.Errorf("subscribing to factop.plugin: %w", err)
	}

	if err := pm.Nats.Subscribe("factorio.stdout", pm.stdoutMonitor); err != nil {
		return fmt.Errorf("subscribing to factorio.stdout: %w", err)
	}

	if err := pm.Nats.Subscribe("factop.plugin.deploy.>", pm.handleDeployChunk); err != nil {
		return fmt.Errorf("subscribing to factop.plugin.deploy: %w", err)
	}

	pm.Info("plugin manager started", "plugins", len(pm.registry.Plugins))
	return nil
}

func (pm *PluginManager) Shutdown() error {
	pm.stopAllPlugins()
	return nil
}

// handleCommand dispatches plugin management commands received on factop.plugin.
func (pm *PluginManager) handleCommand(msg *nats.Msg) {
	var parts []string

	// Support deploy commands sent via msg.Header (for binary payload deploys).
	// Check header first so binary payloads aren't misinterpreted as text commands.
	if msg.Header != nil {
		if cmd := msg.Header.Get("command"); cmd != "" {
			parts = strings.Fields(cmd)
		}
	}
	if len(parts) == 0 {
		parts = strings.Fields(string(msg.Data))
	}

	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "register":
		pm.cmdRegister(msg, parts[1:])
	case "unregister":
		pm.cmdUnregister(msg, parts[1:])
	case "start":
		pm.cmdStart(msg, parts[1:])
	case "stop":
		pm.cmdStop(msg, parts[1:])
	case "restart":
		pm.cmdRestart(msg, parts[1:])
	case "status":
		pm.cmdStatus(msg)
	case "deploy":
		pm.cmdDeploy(msg, parts[1:])
	case "rollback":
		pm.cmdRollback(msg, parts[1:])
	case "versions":
		pm.cmdVersions(msg, parts[1:])
	case "version-remove":
		pm.cmdVersionRemove(msg, parts[1:])
	case "list":
		pm.cmdList(msg)
	default:
		pm.Nats.Reply(msg, nil, errors.New("unknown plugin command: "+parts[0]))
	}
}

// stdoutMonitor watches factorio.stdout for the RCON startup marker to trigger
// auto-start of enabled plugins, and detects server shutdown to stop plugins.
func (pm *PluginManager) stdoutMonitor(msg *nats.Msg) {
	line := string(msg.Data)

	if strings.Contains(line, RconStartupMarker) {
		if IgnorePattern.MatchString(line) {
			return
		}
		pm.Info("plugin manager detected RCON startup")
		// The Factorio component also handles this message and transitions
		// to StateRunning. Since NATS delivery order between subscribers is
		// not guaranteed, poll briefly for the state transition.
		go func() {
			deadline := time.After(5 * time.Second)
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-deadline:
					pm.Warn("timed out waiting for Factorio StateRunning, starting plugins anyway")
					pm.autoStartPlugins()
					return
				case <-ticker.C:
					pm.Factorio.mu.Lock()
					state := pm.Factorio.state
					pm.Factorio.mu.Unlock()
					if state == StateRunning {
						pm.autoStartPlugins()
						return
					}
				}
			}
		}()
	}
}

// autoStartPlugins starts all enabled plugins in registration order.
func (pm *PluginManager) autoStartPlugins() {
	pm.mu.Lock()
	plugins := make([]PluginEntry, len(pm.registry.Plugins))
	copy(plugins, pm.registry.Plugins)
	pm.mu.Unlock()

	for _, p := range plugins {
		if !p.Enabled {
			continue
		}
		if err := pm.startPlugin(p.Name); err != nil {
			pm.Error("auto-start plugin failed", "name", p.Name, "error", err)
		}
	}
}

// stopAllPlugins stops all running plugins in reverse registration order.
func (pm *PluginManager) stopAllPlugins() {
	pm.mu.Lock()
	plugins := make([]PluginEntry, len(pm.registry.Plugins))
	copy(plugins, pm.registry.Plugins)
	pm.mu.Unlock()

	// Iterate in reverse registration order.
	for i := len(plugins) - 1; i >= 0; i-- {
		p := plugins[i]
		pm.mu.Lock()
		inst, ok := pm.instances[p.Name]
		pm.mu.Unlock()

		if !ok {
			continue
		}

		inst.mu.Lock()
		state := inst.state
		inst.mu.Unlock()

		if state == PluginRunning || state == PluginStarting {
			if err := pm.stopPlugin(p.Name); err != nil {
				pm.Error("stop plugin failed during shutdown", "name", p.Name, "error", err)
			}
		}
	}
}

// startPlugin starts a plugin process by name.
func (pm *PluginManager) startPlugin(name string) error {
	pm.mu.Lock()
	entry := pm.registry.Find(name)
	pm.mu.Unlock()

	if entry == nil {
		return fmt.Errorf("plugin not found: %s", name)
	}

	pm.mu.Lock()
	inst, ok := pm.instances[name]
	if !ok {
		inst = &pluginInstance{entry: *entry}
		pm.instances[name] = inst
	}
	pm.mu.Unlock()

	inst.mu.Lock()
	if err := inst.transition(PluginStarting); err != nil {
		inst.mu.Unlock()
		return err
	}

	dataDir := filepath.Join(PluginBaseDir, "data", name)
	cmd := exec.Command(entry.BinaryPath,
		"--nats-url", "nats://localhost:4222",
		"--data-dir", dataDir+"/",
		"--plugin-name", name,
	)
	// Pipe plugin stdout and stderr to NATS for observability.
	subject := "plugin." + name
	cmd.Stdout = &natsWriter{nc: pm.Nats, subject: subject}
	cmd.Stderr = &natsWriter{nc: pm.Nats, subject: subject}

	if err := cmd.Start(); err != nil {
		inst.state = PluginErrored
		inst.mu.Unlock()
		return fmt.Errorf("starting plugin %s: %w", name, err)
	}

	inst.cmd = cmd
	inst.pid = cmd.Process.Pid
	inst.entry = *entry
	inst.restartCount = 0
	inst.done = make(chan struct{})
	inst.mu.Unlock()

	go pm.processMonitor(inst)
	go pm.healthCheck(name)

	pm.Info("plugin starting", "name", name, "version", entry.Version)
	return nil
}

// stopPlugin stops a running plugin process by name.
func (pm *PluginManager) stopPlugin(name string) error {
	pm.mu.Lock()
	inst, ok := pm.instances[name]
	pm.mu.Unlock()

	if !ok {
		return fmt.Errorf("plugin instance not found: %s", name)
	}

	inst.mu.Lock()
	if err := inst.transition(PluginStopping); err != nil {
		inst.mu.Unlock()
		return err
	}
	done := inst.done
	version := inst.entry.Version
	inst.mu.Unlock()

	// Send SIGINT for graceful shutdown.
	if inst.cmd != nil && inst.cmd.Process != nil {
		_ = inst.cmd.Process.Signal(syscall.SIGINT)
	}

	// Wait up to 10 seconds for the process to exit.
	select {
	case <-done:
		// Process exited gracefully.
	case <-time.After(10 * time.Second):
		// Timeout — send SIGKILL.
		if inst.cmd != nil && inst.cmd.Process != nil {
			_ = inst.cmd.Process.Kill()
		}
		<-done
	}

	inst.mu.Lock()
	_ = inst.transition(PluginStopped)
	inst.mu.Unlock()

	pm.Info("plugin stopped", "name", name, "version", version)
	return nil
}

// restartPlugin stops and then starts a plugin by name.
func (pm *PluginManager) restartPlugin(name string) error {
	if err := pm.stopPlugin(name); err != nil {
		return fmt.Errorf("stopping plugin for restart: %w", err)
	}
	return pm.startPlugin(name)
}

// processMonitor waits for a plugin process to exit and handles restart logic.
func (pm *PluginManager) processMonitor(inst *pluginInstance) {
	err := inst.cmd.Wait()

	inst.mu.Lock()
	state := inst.state
	name := inst.entry.Name
	version := inst.entry.Version
	inst.mu.Unlock()

	if state == PluginStopping {
		// Expected shutdown — just close done and return.
		close(inst.done)
		return
	}

	// Unexpected exit — transition to Errored.
	inst.mu.Lock()
	_ = inst.transition(PluginErrored)
	inst.mu.Unlock()

	exitCode := -1
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}
	pm.Error("plugin exited unexpectedly", "name", name, "version", version, "exitCode", exitCode)

	close(inst.done)

	// Attempt restart with exponential backoff: 2s, 4s, 8s (max 3 attempts).
	backoffs := []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second}
	for attempt, delay := range backoffs {
		pm.Info("plugin restart attempt", "name", name, "attempt", attempt+1, "delay", delay)
		time.Sleep(delay)
		if err := pm.startPlugin(name); err != nil {
			pm.Error("plugin restart failed", "name", name, "attempt", attempt+1, "error", err)
			continue
		}
		// Restart succeeded.
		return
	}

	pm.Error("plugin restart exhausted all attempts", "name", name, "version", version)
}

// healthCheck periodically pings a plugin on plugin.<name>.health and manages
// the Starting→Running transition on first success. After 3 consecutive missed
// checks the plugin is killed and the processMonitor handles restart.
func (pm *PluginManager) healthCheck(name string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	consecutiveFailures := 0

	for {
		select {
		case <-pm.Context.Done():
			return
		case <-ticker.C:
			pm.mu.Lock()
			inst, ok := pm.instances[name]
			pm.mu.Unlock()
			if !ok {
				return
			}

			inst.mu.Lock()
			state := inst.state
			inst.mu.Unlock()

			if state != PluginRunning && state != PluginStarting {
				return
			}

			// Send health check request with a 5-second timeout.
			resp, err := pm.Nats.conn.Request("plugin."+name+".health", nil, 5*time.Second)
			if err != nil {
				consecutiveFailures++
				pm.Warn("health check failed", "name", name, "consecutive", consecutiveFailures)

				if consecutiveFailures >= 3 {
					// Terminate and let processMonitor handle restart.
					inst.mu.Lock()
					_ = inst.transition(PluginErrored)
					inst.mu.Unlock()
					if inst.cmd != nil && inst.cmd.Process != nil {
						_ = inst.cmd.Process.Kill()
					}
					pm.Error("health check threshold exceeded, killing plugin", "name", name)
					return
				}
				continue
			}

			// Health check succeeded.
			_ = resp // response payload not used by manager
			consecutiveFailures = 0

			inst.mu.Lock()
			inst.lastSeen = time.Now()
			if inst.state == PluginStarting {
				_ = inst.transition(PluginRunning)
				pm.Info("plugin running", "name", name, "version", inst.entry.Version, "pid", inst.pid)
			}
			inst.mu.Unlock()
		}
	}
}

// cmdRegister handles the "register <name> <binary-path>" command.
func (pm *PluginManager) cmdRegister(msg *nats.Msg, args []string) {
	if len(args) < 2 {
		pm.Nats.Reply(msg, nil, errors.New("usage: register <name> <binary-path>"))
		return
	}
	name := args[0]
	binaryPath := args[1]

	version := "0.0.0"
	// Attempt to extract version from binary path (e.g. .../bin/name/1.0.0/name).
	dir := filepath.Dir(binaryPath)
	if v := filepath.Base(dir); v != "." && v != "/" {
		version = v
	}

	entry := PluginEntry{
		Name:       name,
		Version:    version,
		BinaryPath: binaryPath,
		Enabled:    true,
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	if err := pm.registry.Add(entry); err != nil {
		pm.Nats.Reply(msg, nil, err)
		return
	}

	dataDir := filepath.Join(PluginBaseDir, "data", name)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		pm.Nats.Reply(msg, nil, fmt.Errorf("creating data dir: %w", err))
		return
	}

	if err := pm.registry.Save(PluginRegistryPath); err != nil {
		pm.Nats.Reply(msg, nil, fmt.Errorf("saving registry: %w", err))
		return
	}

	pm.Info("plugin registered", "name", name, "version", version)
	pm.Nats.Reply(msg, []byte("registered "+name), nil)
}

// cmdUnregister handles the "unregister <name>" command.
func (pm *PluginManager) cmdUnregister(msg *nats.Msg, args []string) {
	if len(args) < 1 {
		pm.Nats.Reply(msg, nil, errors.New("usage: unregister <name>"))
		return
	}
	name := args[0]

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Stop the plugin if it is currently running.
	if _, ok := pm.instances[name]; ok {
		pm.mu.Unlock()
		_ = pm.stopPlugin(name)
		pm.mu.Lock()
		delete(pm.instances, name)
	}

	if _, err := pm.registry.Remove(name); err != nil {
		pm.Nats.Reply(msg, nil, err)
		return
	}

	if err := pm.registry.Save(PluginRegistryPath); err != nil {
		pm.Nats.Reply(msg, nil, fmt.Errorf("saving registry: %w", err))
		return
	}

	dataDir := filepath.Join(PluginBaseDir, "data", name)
	binDir := filepath.Join(PluginBaseDir, "bin", name)
	if err := os.RemoveAll(binDir); err != nil {
		pm.Warn("failed to remove plugin bin dir", "name", name, "error", err)
	}
	pm.Info("plugin unregistered, data dir preserved", "name", name, "dataDir", dataDir)
	pm.Nats.Reply(msg, []byte("unregistered "+name), nil)
}

// cmdStart handles the "start <name>" command.
func (pm *PluginManager) cmdStart(msg *nats.Msg, args []string) {
	if len(args) < 1 {
		pm.Nats.Reply(msg, nil, errors.New("usage: start <name>"))
		return
	}
	name := args[0]

	pm.mu.Lock()
	entry := pm.registry.Find(name)
	pm.mu.Unlock()

	if entry == nil {
		pm.Nats.Reply(msg, nil, fmt.Errorf("plugin not found: %s", name))
		return
	}

	if err := pm.startPlugin(name); err != nil {
		pm.Nats.Reply(msg, nil, err)
		return
	}
	pm.Nats.Reply(msg, []byte("starting "+name), nil)
}

// cmdStop handles the "stop <name>" command.
func (pm *PluginManager) cmdStop(msg *nats.Msg, args []string) {
	if len(args) < 1 {
		pm.Nats.Reply(msg, nil, errors.New("usage: stop <name>"))
		return
	}
	name := args[0]

	pm.mu.Lock()
	entry := pm.registry.Find(name)
	pm.mu.Unlock()

	if entry == nil {
		pm.Nats.Reply(msg, nil, fmt.Errorf("plugin not found: %s", name))
		return
	}

	if err := pm.stopPlugin(name); err != nil {
		pm.Nats.Reply(msg, nil, err)
		return
	}
	pm.Nats.Reply(msg, []byte("stopping "+name), nil)
}

// cmdRestart handles the "restart <name>" command.
func (pm *PluginManager) cmdRestart(msg *nats.Msg, args []string) {
	if len(args) < 1 {
		pm.Nats.Reply(msg, nil, errors.New("usage: restart <name>"))
		return
	}
	name := args[0]

	pm.mu.Lock()
	entry := pm.registry.Find(name)
	pm.mu.Unlock()

	if entry == nil {
		pm.Nats.Reply(msg, nil, fmt.Errorf("plugin not found: %s", name))
		return
	}

	if err := pm.restartPlugin(name); err != nil {
		pm.Nats.Reply(msg, nil, err)
		return
	}
	pm.Nats.Reply(msg, []byte("restarting "+name), nil)
}

// cmdStatus handles the "status" command.
// Format: name:version:state:pid,...
func (pm *PluginManager) cmdStatus(msg *nats.Msg) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var entries []string
	for _, p := range pm.registry.Plugins {
		state := PluginStopped.String()
		pid := 0
		version := p.Version

		if inst, ok := pm.instances[p.Name]; ok {
			inst.mu.Lock()
			state = inst.state.String()
			pid = inst.pid
			inst.mu.Unlock()
		}

		entries = append(entries, p.Name+":"+version+":"+state+":"+strconv.Itoa(pid))
	}

	pm.Nats.Reply(msg, []byte(strings.Join(entries, ",")), nil)
}

// cmdDeploy handles the "deploy <name> <version>" command.
// Binary data is in msg.Data. The command may come from msg.Header or text args.
func (pm *PluginManager) cmdDeploy(msg *nats.Msg, args []string) {
	if len(args) < 2 {
		pm.Nats.Reply(msg, nil, errors.New("usage: deploy <name> <version>"))
		return
	}
	name := args[0]
	version := args[1]

	binDir := filepath.Join(PluginBaseDir, "bin", name, version)
	binPath := filepath.Join(binDir, name)

	// Reject duplicate versions.
	if _, err := os.Stat(binPath); err == nil {
		pm.Nats.Reply(msg, nil, fmt.Errorf("version %s already exists for %s", version, name))
		return
	}

	if err := os.MkdirAll(binDir, 0755); err != nil {
		pm.Nats.Reply(msg, nil, fmt.Errorf("creating bin dir: %w", err))
		return
	}

	if err := os.WriteFile(binPath, msg.Data, 0755); err != nil {
		pm.Nats.Reply(msg, nil, fmt.Errorf("writing binary: %w", err))
		return
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	entry := pm.registry.Find(name)
	if entry != nil {
		// Plugin is registered — check if running.
		inst, running := pm.instances[name]
		if running {
			inst.mu.Lock()
			state := inst.state
			inst.mu.Unlock()

			if state == PluginRunning || state == PluginStarting {
				pm.mu.Unlock()
				_ = pm.stopPlugin(name)
				pm.mu.Lock()
			}
		}
		_ = pm.registry.UpdateVersion(name, version, binPath)
		if err := pm.registry.Save(PluginRegistryPath); err != nil {
			pm.Nats.Reply(msg, nil, fmt.Errorf("saving registry: %w", err))
			return
		}
		if running {
			inst.mu.Lock()
			state := inst.state
			inst.mu.Unlock()
			if state == PluginStopped || state == PluginErrored {
				pm.mu.Unlock()
				_ = pm.startPlugin(name)
				pm.mu.Lock()
			}
		}
	} else {
		// Plugin not registered — register it.
		newEntry := PluginEntry{
			Name:       name,
			Version:    version,
			BinaryPath: binPath,
			Enabled:    true,
		}
		if err := pm.registry.Add(newEntry); err != nil {
			pm.Nats.Reply(msg, nil, err)
			return
		}
		dataDir := filepath.Join(PluginBaseDir, "data", name)
		_ = os.MkdirAll(dataDir, 0755)
		if err := pm.registry.Save(PluginRegistryPath); err != nil {
			pm.Nats.Reply(msg, nil, fmt.Errorf("saving registry: %w", err))
			return
		}
	}

	pm.Info("plugin deployed", "name", name, "version", version)
	pm.Nats.Reply(msg, []byte("deployed "+name+" "+version), nil)
}

// cmdRollback handles the "rollback <name> <version>" command.
func (pm *PluginManager) cmdRollback(msg *nats.Msg, args []string) {
	if len(args) < 2 {
		pm.Nats.Reply(msg, nil, errors.New("usage: rollback <name> <version>"))
		return
	}
	name := args[0]
	version := args[1]

	binPath := filepath.Join(PluginBaseDir, "bin", name, version, name)
	if _, err := os.Stat(binPath); err != nil {
		pm.Nats.Reply(msg, nil, fmt.Errorf("version %s not found for %s", version, name))
		return
	}

	pm.mu.Lock()
	inst, hasInst := pm.instances[name]
	pm.mu.Unlock()

	// Stop current version if running.
	if hasInst {
		inst.mu.Lock()
		state := inst.state
		inst.mu.Unlock()
		if state == PluginRunning || state == PluginStarting {
			_ = pm.stopPlugin(name)
		}
	}

	pm.mu.Lock()
	_ = pm.registry.UpdateVersion(name, version, binPath)
	if err := pm.registry.Save(PluginRegistryPath); err != nil {
		pm.mu.Unlock()
		pm.Nats.Reply(msg, nil, fmt.Errorf("saving registry: %w", err))
		return
	}
	pm.mu.Unlock()

	// Start the target version.
	if err := pm.startPlugin(name); err != nil {
		pm.Nats.Reply(msg, nil, fmt.Errorf("starting rolled-back version: %w", err))
		return
	}

	pm.Info("plugin rolled back", "name", name, "version", version)
	pm.Nats.Reply(msg, []byte("rolled back "+name+" to "+version), nil)
}

// cmdVersions handles the "versions <name>" command.
// Format: current:<version>:state:<state>,installed:<v1>,<v2>,...
func (pm *PluginManager) cmdVersions(msg *nats.Msg, args []string) {
	if len(args) < 1 {
		pm.Nats.Reply(msg, nil, errors.New("usage: versions <name>"))
		return
	}
	name := args[0]

	pm.mu.Lock()
	entry := pm.registry.Find(name)
	var currentVersion string
	if entry != nil {
		currentVersion = entry.Version
	}

	state := PluginStopped.String()
	if inst, ok := pm.instances[name]; ok {
		inst.mu.Lock()
		state = inst.state.String()
		inst.mu.Unlock()
	}
	pm.mu.Unlock()

	versions, err := installedVersions(name)
	if err != nil {
		pm.Nats.Reply(msg, nil, fmt.Errorf("reading versions: %w", err))
		return
	}

	result := "current:" + currentVersion + ":state:" + state + ",installed:" + strings.Join(versions, ",")
	pm.Nats.Reply(msg, []byte(result), nil)
}

// cmdVersionRemove handles the "version-remove <name> <version>" command.
func (pm *PluginManager) cmdVersionRemove(msg *nats.Msg, args []string) {
	if len(args) < 2 {
		pm.Nats.Reply(msg, nil, errors.New("usage: version-remove <name> <version>"))
		return
	}
	name := args[0]
	version := args[1]

	pm.mu.Lock()
	entry := pm.registry.Find(name)
	pm.mu.Unlock()

	// Reject if this is the currently active version.
	if entry != nil && entry.Version == version {
		pm.Nats.Reply(msg, nil, fmt.Errorf("cannot remove active version %s for %s", version, name))
		return
	}

	versionDir := filepath.Join(PluginBaseDir, "bin", name, version)
	if err := os.RemoveAll(versionDir); err != nil {
		pm.Nats.Reply(msg, nil, fmt.Errorf("removing version dir: %w", err))
		return
	}

	pm.Info("plugin version removed", "name", name, "version", version)
	pm.Nats.Reply(msg, []byte("removed "+name+" "+version), nil)
}

// cmdList handles the "list" command.
// Format: name:version:state:installed_count,...
func (pm *PluginManager) cmdList(msg *nats.Msg) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var entries []string
	for _, p := range pm.registry.Plugins {
		state := PluginStopped.String()

		if inst, ok := pm.instances[p.Name]; ok {
			inst.mu.Lock()
			state = inst.state.String()
			inst.mu.Unlock()
		}

		installedCount := 0
		versions, err := installedVersions(p.Name)
		if err == nil {
			installedCount = len(versions)
		}

		entries = append(entries, p.Name+":"+p.Version+":"+state+":"+strconv.Itoa(installedCount))
	}

	pm.Nats.Reply(msg, []byte(strings.Join(entries, ",")), nil)
}

// installedVersions returns the list of version directories for a plugin.
func installedVersions(name string) ([]string, error) {
	binDir := filepath.Join(PluginBaseDir, "bin", name)
	entries, err := os.ReadDir(binDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var versions []string
	for _, e := range entries {
		if e.IsDir() {
			versions = append(versions, e.Name())
		}
	}
	return versions, nil
}

// handleDeployChunk handles messages on factop.plugin.deploy.<name>.<version>
// for chunked binary deploys. Chunks arrive with "chunk-index" and "chunk-total"
// headers. Once all chunks are received, the binary is reassembled and passed
// to cmdDeploy. If no chunk headers are present, the message is treated as a
// single-message deploy and forwarded directly.
func (pm *PluginManager) handleDeployChunk(msg *nats.Msg) {
	// Subject format: factop.plugin.deploy.<name>.<version>
	// Version may contain dots (e.g. 0.2.0), so join all tokens from index 4.
	parts := strings.Split(msg.Subject, ".")
	if len(parts) < 5 {
		pm.Nats.Reply(msg, nil, errors.New("invalid deploy subject"))
		return
	}
	name := parts[3]
	version := strings.Join(parts[4:], ".")

	// No chunk headers — treat as single-message deploy.
	idxStr := msg.Header.Get("chunk-index")
	totalStr := msg.Header.Get("chunk-total")
	if idxStr == "" || totalStr == "" {
		pm.cmdDeploy(msg, []string{name, version})
		return
	}

	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		pm.Nats.Reply(msg, nil, fmt.Errorf("invalid chunk-index: %w", err))
		return
	}
	total, err := strconv.Atoi(totalStr)
	if err != nil {
		pm.Nats.Reply(msg, nil, fmt.Errorf("invalid chunk-total: %w", err))
		return
	}

	key := name + "/" + version

	pm.mu.Lock()
	buf, ok := pm.chunks[key]
	if !ok {
		buf = &deployBuffer{total: total, received: make(map[int][]byte)}
		pm.chunks[key] = buf
	}
	// Copy chunk data so the NATS message buffer can be reused.
	chunk := make([]byte, len(msg.Data))
	copy(chunk, msg.Data)
	buf.received[idx] = chunk

	if len(buf.received) < buf.total {
		// Not all chunks received yet — ack this chunk and wait.
		pm.mu.Unlock()
		pm.Nats.Reply(msg, []byte(fmt.Sprintf("chunk %d/%d received", idx+1, total)), nil)
		return
	}

	// All chunks received — reassemble in order.
	delete(pm.chunks, key)
	pm.mu.Unlock()

	var assembled []byte
	for i := 0; i < buf.total; i++ {
		assembled = append(assembled, buf.received[i]...)
	}

	// Create a synthetic message with the full binary for cmdDeploy.
	deployMsg := nats.NewMsg(msg.Subject)
	deployMsg.Reply = msg.Reply
	deployMsg.Data = assembled

	pm.Info("chunked deploy reassembled", "name", name, "version", version,
		"chunks", buf.total, "bytes", len(assembled))
	pm.cmdDeploy(deployMsg, []string{name, version})
}

// natsWriter implements io.Writer by publishing each complete line
// as a separate NATS message. Partial lines are buffered until a
// newline is received.
type natsWriter struct {
	nc      *Nats
	subject string
	buf     []byte
}

func (w *natsWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		idx := bytes.IndexByte(w.buf, '\n')
		if idx < 0 {
			break
		}
		line := w.buf[:idx]
		if len(line) > 0 {
			w.nc.Publish(w.subject, line)
		}
		w.buf = w.buf[idx+1:]
	}
	return len(p), nil
}
