package service

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/mlctrez/bind"
	"github.com/nats-io/nats.go"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

var _ bind.Startup = (*Factorio)(nil)
var _ bind.Shutdown = (*Factorio)(nil)

type Factorio struct {
	slog.Logger
	Nats    *Nats
	SoftMod *SoftMod

	cmd        *exec.Cmd
	stdErrPipe io.ReadCloser
	stdOutPipe io.ReadCloser
	stdInPipe  io.WriteCloser

	Port         int
	RconBind     string
	RconPassword string
}

const FactorioDirectory = "/opt/factorio/current"
const FactorioBinary = FactorioDirectory + "/bin/x64/factorio"
const SaveLocation = "/opt/factorio/saves"
const SaveFile = SaveLocation + "/save.zip"
const CreateSaveOutput = SaveLocation + "/create-save.log"

func (f *Factorio) Reset() error {
	if f.Status() == "running" {
		if err := f.Shutdown(); err != nil {
			return err
		}
	}
	if err := os.Remove(SaveFile); err != nil {
		return err
	}
	if err := f.saveFileSetup(); err != nil {
		return err
	}
	return f.Restart()
}

func (f *Factorio) saveFileSetup() error {
	if err := os.MkdirAll(SaveLocation, 0755); err != nil {
		return err
	}
	if _, err := os.Stat(SaveFile); os.IsNotExist(err) {
		// TODO: add map gen settings
		gen := exec.Command(FactorioBinary,
			//"--map-gen-settings", MapGenSettings,
			"--create", SaveFile,
		)
		var output []byte
		output, err = gen.CombinedOutput()
		if createErr := os.WriteFile(CreateSaveOutput, output, 0644); createErr != nil {
			return fmt.Errorf("factorio save file create error: %v ", createErr)
		}
		if err != nil {
			return fmt.Errorf("factorio save file execute error: %v", err)
		}

		if _, err = os.Stat(CurrentSoftMod); os.IsNotExist(err) {
			return nil
		}

		var payload []byte
		if payload, err = os.ReadFile(CurrentSoftMod); err != nil {
			return err
		}
		if err = f.SoftMod.Apply(payload); err != nil {
			return err
		}
	}
	return nil
}

func (f *Factorio) cmdSetup() error {
	// TODO: read these from settings
	f.Port = 34198
	f.RconBind = "127.0.0.1:3000"
	f.RconPassword = strings.ReplaceAll(uuid.NewString(), "-", "")
	f.cmd = exec.Command(
		FactorioBinary,
		"--enable-lua-udp", "4001",
		"--port", fmt.Sprintf("%d", f.Port),
		"--start-server", SaveFile,
		"--rcon-bind", f.RconBind,
		"--rcon-password", f.RconPassword,
		"--server-settings", ServerSettings,
		"--server-adminlist", ServerAdminList,
		"--server-banlist", ServerBanList,
	)
	f.cmd.Dir = FactorioDirectory
	var err error
	if f.stdErrPipe, err = f.cmd.StderrPipe(); err != nil {
		return err
	}
	if f.stdOutPipe, err = f.cmd.StdoutPipe(); err != nil {
		return err
	}
	if f.stdInPipe, err = f.cmd.StdinPipe(); err != nil {
		return err
	}
	go f.publisher("stderr", f.stdErrPipe)
	go f.publisher("stdout", f.stdOutPipe)

	return nil
}

func (f *Factorio) Startup() error {
	if err := f.saveFileSetup(); err != nil {
		return err
	}
	if err := f.cmdSetup(); err != nil {
		return err
	}
	if err := f.Nats.Subscribe("factorio.stdin", f.stdinHandler); err != nil {
		return err
	}
	if err := f.Nats.Subscribe("factorio.softmod", f.softmodHandler); err != nil {
		return err
	}
	f.Info("starting factorio")
	return f.cmd.Start()
}

func (f *Factorio) Restart() error {
	f.Info("restarting factorio")
	if f.cmd.Process != nil && f.cmd.ProcessState == nil {
		if err := f.cmd.Process.Signal(os.Interrupt); err != nil {
			return err
		}
		if err := f.cmd.Wait(); err != nil {
			return err
		}
	}
	if err := f.cmdSetup(); err != nil {
		return err
	}
	return f.cmd.Start()
}

func (f *Factorio) Shutdown() error {

	if f.cmd.Process != nil && f.cmd.ProcessState == nil {
		if err := f.cmd.Process.Signal(os.Interrupt); err != nil {
			f.Error("factorio interrupt error", "error", err)
			return err
		}
		if err := f.cmd.Wait(); err != nil {
			var exitError *exec.ExitError
			if errors.As(err, &exitError) {
				return nil
			}
			f.Error("factorio wait error", "error", err)
			return err
		}
		if f.stdInPipe != nil {
			if err := f.stdInPipe.Close(); err != nil {
				if !errors.Is(err, os.ErrClosed) {
					f.Error("factorio stdin pipe error", "error", err)
				}
			}
			f.stdInPipe = nil
		}
	}
	return nil
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
		// TODO: revisit logging format
		switch pipeName {
		case "stderr":
			f.Error(line)
		default:
			f.Info(line)
		}
		//f.Info("factorio message", "pipeName", pipeName, "line", line)

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
			f.Info("factorio.stdin %s", logMessage)
		}
		_, _ = f.stdInPipe.Write(msg.Data)
		_, _ = f.stdInPipe.Write([]byte(NewLine))
	}
}

func (f *Factorio) Status() string {
	if f.cmd.ProcessState == nil {
		return "running"
	}
	if f.cmd.ProcessState.Exited() {
		return "stopped"
	}
	return "unknown"
}

func (f *Factorio) softmodHandler(msg *nats.Msg) {
	if f.Status() == "running" {
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
