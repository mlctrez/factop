package service

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mlctrez/bind"
	"github.com/nats-io/nats.go"
)

const DefaultShutdownTimeout = 30 * time.Second

// ServerState represents the lifecycle state of the Factorio server process.
type ServerState int

const (
	StateStopped  ServerState = iota // No process running
	StateStarting                    // cmd.Start() called, waiting for RCON marker
	StateRunning                     // RCON marker detected, server accepting commands
	StateStopping                    // Interrupt sent, waiting for process exit
	StateError                       // Unexpected process exit
)

func (s ServerState) String() string {
	switch s {
	case StateStopped:
		return "stopped"
	case StateStarting:
		return "starting"
	case StateRunning:
		return "running"
	case StateStopping:
		return "stopping"
	case StateError:
		return "error"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}

// validTransitions defines the allowed state transitions.
var validTransitions = map[ServerState]map[ServerState]bool{
	StateStopped:  {StateStarting: true},
	StateError:    {StateStarting: true},
	StateStarting: {StateRunning: true, StateError: true, StateStopping: true},
	StateRunning:  {StateStopping: true, StateError: true},
	StateStopping: {StateStopped: true},
}

var _ bind.Startup = (*Factorio)(nil)
var _ bind.Shutdown = (*Factorio)(nil)

type Factorio struct {
	slog.Logger
	Nats     *Nats
	SoftMod  *SoftMod
	Settings *Settings

	mu    sync.Mutex
	state ServerState
	done  chan struct{} // closed by processMonitor when the process exits

	cmd        *exec.Cmd
	stdErrPipe io.ReadCloser
	stdOutPipe io.ReadCloser
	stdInPipe  io.WriteCloser

	Port         int
	RconBind     string
	RconPassword string
}

// transition validates and performs a state transition. Must be called with f.mu held.
func (f *Factorio) transition(to ServerState) error {
	from := f.state
	if targets, ok := validTransitions[from]; ok && targets[to] {
		f.state = to
		f.Info("state transition", "from", from.String(), "to", to.String())
		return nil
	}
	return fmt.Errorf("invalid state transition from %s to %s", from.String(), to.String())
}

func (f *Factorio) binary() string {
	return fmt.Sprintf("/opt/factorio/%s/bin/x64/factorio", f.Settings.Data.FactorioVersion)
}

func (f *Factorio) directory() string {
	return fmt.Sprintf("/opt/factorio/%s", f.Settings.Data.FactorioVersion)
}

const SaveLocation = "/opt/factorio/saves"
const SaveFile = SaveLocation + "/save.zip"
const CreateSaveOutput = SaveLocation + "/create-save.log"

func (f *Factorio) Reset() error {
	f.mu.Lock()
	isRunning := f.state == StateRunning || f.state == StateStarting
	f.mu.Unlock()
	if isRunning {
		if err := f.Shutdown(); err != nil {
			return fmt.Errorf("stopping server for reset: %w", err)
		}
	}
	if err := os.Remove(SaveFile); err != nil {
		return fmt.Errorf("removing save file: %w", err)
	}
	if err := f.saveFileSetup(); err != nil {
		return fmt.Errorf("setting up save file: %w", err)
	}
	return f.Restart()
}

func (f *Factorio) saveFileSetup() error {
	if err := os.MkdirAll(SaveLocation, 0755); err != nil {
		return fmt.Errorf("creating save directory: %w", err)
	}
	if _, err := os.Stat(SaveFile); os.IsNotExist(err) {
		// TODO: add map gen settings
		gen := exec.Command(f.binary(),
			//"--map-gen-settings", MapGenSettings,
			"--create", SaveFile,
		)
		var output []byte
		output, err = gen.CombinedOutput()
		if createErr := os.WriteFile(CreateSaveOutput, output, 0644); createErr != nil {
			return fmt.Errorf("writing save creation log: %w", createErr)
		}
		if err != nil {
			return fmt.Errorf("executing save file creation: %w", err)
		}

		if _, err = os.Stat(CurrentSoftMod); os.IsNotExist(err) {
			return nil
		}

		var payload []byte
		if payload, err = os.ReadFile(CurrentSoftMod); err != nil {
			return fmt.Errorf("reading current softmod: %w", err)
		}
		if err = f.SoftMod.Apply(payload); err != nil {
			return fmt.Errorf("applying softmod to new save: %w", err)
		}
	}
	return nil
}

func (f *Factorio) cmdSetup() error {
	f.Port = f.Settings.Data.GamePort
	f.RconBind = f.Settings.Data.RconBind
	f.RconPassword = strings.ReplaceAll(uuid.NewString(), "-", "")
	f.done = make(chan struct{})
	f.cmd = exec.Command(
		f.binary(),
		"--enable-lua-udp", fmt.Sprintf("%d", f.Settings.Data.UDPOutgoing),
		"--port", fmt.Sprintf("%d", f.Port),
		"--start-server", SaveFile,
		"--rcon-bind", f.RconBind,
		"--rcon-password", f.RconPassword,
		"--server-settings", ServerSettings,
		"--server-adminlist", ServerAdminList,
		"--server-banlist", ServerBanList,
	)
	f.cmd.Dir = f.directory()
	var err error
	if f.stdErrPipe, err = f.cmd.StderrPipe(); err != nil {
		return fmt.Errorf("creating stderr pipe: %w", err)
	}
	if f.stdOutPipe, err = f.cmd.StdoutPipe(); err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}
	if f.stdInPipe, err = f.cmd.StdinPipe(); err != nil {
		return fmt.Errorf("creating stdin pipe: %w", err)
	}
	go f.publisher("stderr", f.stdErrPipe)
	go f.publisher("stdout", f.stdOutPipe)

	f.mu.Lock()
	err = f.transition(StateStarting)
	f.mu.Unlock()
	if err != nil {
		return err
	}

	return nil
}

// startProcess runs cmdSetup, starts the process, and launches the monitor goroutine.
// The caller must ensure the state machine is in Stopped or Error before calling.
func (f *Factorio) startProcess() error {
	if err := f.cmdSetup(); err != nil {
		return fmt.Errorf("command setup: %w", err)
	}
	if err := f.cmd.Start(); err != nil {
		f.mu.Lock()
		f.state = StateError
		f.mu.Unlock()
		return fmt.Errorf("starting factorio process: %w", err)
	}
	go f.processMonitor()
	return nil
}

func (f *Factorio) Startup() error {
	if err := f.saveFileSetup(); err != nil {
		return fmt.Errorf("save file setup: %w", err)
	}
	if err := f.startProcess(); err != nil {
		return err
	}
	if err := f.Nats.Subscribe("factorio.stdin", f.stdinHandler); err != nil {
		return fmt.Errorf("subscribing to factorio.stdin: %w", err)
	}
	if err := f.Nats.Subscribe("factorio.softmod", f.softmodHandler); err != nil {
		return fmt.Errorf("subscribing to factorio.softmod: %w", err)
	}
	if err := f.Nats.Subscribe("factorio.stdout", f.stdoutStateMonitor); err != nil {
		return fmt.Errorf("subscribing to factorio.stdout: %w", err)
	}
	f.Info("starting factorio")
	return nil
}

func (f *Factorio) Start() error {
	f.mu.Lock()
	canStart := f.state == StateStopped || f.state == StateError
	f.mu.Unlock()
	if !canStart {
		return fmt.Errorf("cannot start: current state is %s", f.Status())
	}
	return f.startProcess()
}

func (f *Factorio) Restart() error {
	f.Info("restarting factorio")

	f.mu.Lock()
	needsShutdown := f.state == StateRunning || f.state == StateStarting
	f.mu.Unlock()

	if needsShutdown {
		if err := f.Shutdown(); err != nil {
			f.Error("shutdown during restart failed", "error", err)
		}
	}

	return f.startProcess()
}

func (f *Factorio) Shutdown() error {
	f.mu.Lock()
	if f.state != StateRunning && f.state != StateStarting {
		f.mu.Unlock()
		return fmt.Errorf("cannot stop: current state is %s", f.state.String())
	}
	if err := f.transition(StateStopping); err != nil {
		f.mu.Unlock()
		return err
	}
	done := f.done
	f.mu.Unlock()

	var errs []error
	if f.cmd.Process != nil {
		if err := f.cmd.Process.Signal(os.Interrupt); err != nil {
			if !errors.Is(err, os.ErrProcessDone) {
				errs = append(errs, fmt.Errorf("sending interrupt signal: %w", err))
			}
		}
	}

	// Wait for the process to exit with a timeout.
	// The processMonitor goroutine will close the done channel and handle
	// the final state transition.
	timeout := DefaultShutdownTimeout
	select {
	case <-done:
		// Process exited, processMonitor handled the transition.
	case <-time.After(timeout):
		f.Info("shutdown timeout reached, sending kill signal")
		if f.cmd.Process != nil {
			if err := f.cmd.Process.Kill(); err != nil {
				if !errors.Is(err, os.ErrProcessDone) {
					errs = append(errs, fmt.Errorf("sending kill signal: %w", err))
				}
			}
		}
		// Wait again briefly for the kill to take effect.
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			errs = append(errs, fmt.Errorf("process did not exit after kill"))
			// Force state to Error so the system isn't stuck in Stopping.
			f.mu.Lock()
			f.state = StateError
			f.mu.Unlock()
		}
	}

	return errors.Join(errs...)
}

func (f *Factorio) publisher(pipeName string, pipe io.ReadCloser) {
	f.Info("factorio publisher entry", "pipeName", pipeName)
	defer func() {
		f.Info("factorio publisher exit", "pipeName", pipeName)
		_ = pipe.Close()
	}()
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		m := nats.NewMsg(fmt.Sprintf("factorio.%s", pipeName))
		line := scanner.Text()
		m.Data = []byte(line)
		f.Nats.PublishMsg(m)
	}
	if err := scanner.Err(); err != nil {
		if !strings.Contains(err.Error(), "file already closed") {
			f.Error("factorio scanner error", "pipeName", pipeName, "error", err)
		}
	}
}

const NewLine = "\n"

func (f *Factorio) stdinHandler(msg *nats.Msg) {
	if f.stdInPipe != nil {
		logMessage := string(msg.Data)
		if len(logMessage) > 100 {
			logMessage = logMessage[:100] + "..."
		}
		if !strings.HasPrefix(logMessage, "/silent-command") {
			f.Info("factorio.stdin", "message", logMessage)
		}
		_, _ = f.stdInPipe.Write(msg.Data)
		_, _ = f.stdInPipe.Write([]byte(NewLine))
	}
}

func (f *Factorio) Status() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state.String()
}

func (f *Factorio) softmodHandler(msg *nats.Msg) {
	f.mu.Lock()
	isRunning := f.state == StateRunning || f.state == StateStarting
	f.mu.Unlock()
	if isRunning {
		f.Info("stopping server to apply softmod")
		if err := f.Shutdown(); err != nil {
			f.Nats.Reply(msg, nil, err)
			return
		}
	}
	f.Info("applying softmod", "length", len(msg.Data))
	if err := f.SoftMod.Apply(msg.Data); err != nil {
		f.Nats.Reply(msg, nil, err)
		return
	}

	f.Info("restarting server after applying softmod")
	err := f.Restart()
	f.Nats.Reply(msg, []byte(f.Status()), err)
}

// processMonitor waits for the Factorio process to exit and transitions the
// state machine accordingly. It should be launched as a goroutine after cmd.Start().
func (f *Factorio) processMonitor() {
	err := f.cmd.Wait()

	// Close stdin pipe after process exits
	if f.stdInPipe != nil {
		if closeErr := f.stdInPipe.Close(); closeErr != nil {
			if !errors.Is(closeErr, os.ErrClosed) {
				f.Error("factorio stdin pipe error", "error", closeErr)
			}
		}
		f.stdInPipe = nil
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	// Signal waiters that the process has exited.
	if f.done != nil {
		close(f.done)
	}

	if f.state == StateStopping {
		// Clean shutdown — transition to Stopped
		if transErr := f.transition(StateStopped); transErr != nil {
			f.Error("process monitor transition error", "error", transErr)
		}
	} else if f.state == StateStarting || f.state == StateRunning {
		// Unexpected exit — transition to Error
		if err != nil {
			f.Error("unexpected process exit", "error", err)
		}
		if transErr := f.transition(StateError); transErr != nil {
			f.Error("process monitor transition error", "error", transErr)
		}
	}
}

// stdoutStateMonitor subscribes to factorio.stdout NATS messages and detects
// the RCON startup marker to transition from Starting to Running.
func (f *Factorio) stdoutStateMonitor(msg *nats.Msg) {
	line := string(msg.Data)
	if strings.Contains(line, RconStartupMarker) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.state == StateStarting {
			if err := f.transition(StateRunning); err != nil {
				f.Error("RCON marker transition error", "error", err)
			}
		}
	}
}
