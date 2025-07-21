package service

import (
	"bytes"
	"context"
	"github.com/kardianos/service"
	"log/slog"
)

var _ slog.Handler = (*LogHandler)(nil)

type LogHandler struct {
	Logger service.Logger
}

func (s *LogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (s *LogHandler) Handle(ctx context.Context, record slog.Record) error {
	msgBuf := &bytes.Buffer{}
	msgBuf.WriteString(record.Level.String() + " " + record.Message)
	record.Attrs(func(attr slog.Attr) bool {
		if !attr.Equal(slog.Attr{}) {
			msgBuf.WriteString(" " + attr.String())
		}
		return true
	})
	switch record.Level {
	case slog.LevelDebug, slog.LevelInfo:
		_ = s.Logger.Info(msgBuf.String())
	case slog.LevelError:
		_ = s.Logger.Error(msgBuf.String())
	}
	return nil
}

func (s *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler { panic("implement me") }
func (s *LogHandler) WithGroup(name string) slog.Handler       { panic("implement me") }
