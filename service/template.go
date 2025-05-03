package service

import (
	"github.com/mlctrez/servicego"
)

var _ Component = (*ComponentTemplate)(nil)

// ComponentTemplate is a template for creating new components
type ComponentTemplate struct {
	servicego.DefaultLogger
}

func (c *ComponentTemplate) Start(s *Service) error {
	c.Logger(s.Log())
	return nil
}

func (c *ComponentTemplate) Stop() error {
	return nil
}
