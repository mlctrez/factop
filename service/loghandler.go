package service

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/kardianos/service"
	"github.com/nats-io/nats.go"
)

var _ slog.Handler = (*LogHandler)(nil)

type LogHandler struct {
	Logger   service.Logger
	mu       sync.RWMutex
	natsConn *nats.Conn
}

func (s *LogHandler) SetNatsConn(conn *nats.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.natsConn = conn
}

func (s *LogHandler) ClearNatsConn() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.natsConn = nil
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
	case slog.LevelError:
		_ = s.Logger.Error(msgBuf.String())
	case slog.LevelWarn:
		_ = s.Logger.Warning(msgBuf.String())
	default:
		_ = s.Logger.Info(msgBuf.String())
	}

	s.mu.RLock()
	conn := s.natsConn
	s.mu.RUnlock()

	if conn != nil {
		attrs := map[string]string{}
		record.Attrs(func(attr slog.Attr) bool {
			if !attr.Equal(slog.Attr{}) {
				attrs[attr.Key] = attr.Value.String()
			}
			return true
		})
		payload := map[string]any{
			"time":  record.Time.Format("2006-01-02T15:04:05.000Z07:00"),
			"level": record.Level.String(),
			"msg":   record.Message,
			"attrs": attrs,
		}
		if data, err := json.Marshal(payload); err == nil {
			_ = conn.Publish("factop.log", data)
		}
	}

	return nil
}

func (s *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return s }
func (s *LogHandler) WithGroup(name string) slog.Handler       { return s }
