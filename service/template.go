package service

import (
	"log/slog"

	"github.com/mlctrez/bind"
)

var _ bind.Startup = (*ComponentTemplate)(nil)
var _ bind.Shutdown = (*ComponentTemplate)(nil)

// ComponentTemplate is a template for creating new components
type ComponentTemplate struct {
	slog.Logger
}

func (c *ComponentTemplate) Startup() error {
	return nil
}

func (c *ComponentTemplate) Shutdown() error {
	return nil
}
