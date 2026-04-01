package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// PluginEntry represents a single plugin in the registry.
type PluginEntry struct {
	Name       string            `json:"name"`
	Version    string            `json:"version"`
	BinaryPath string            `json:"binary_path"`
	Enabled    bool              `json:"enabled"`
	Settings   map[string]string `json:"settings,omitempty"`
}

// PluginRegistry is the persistent list of installed plugins.
type PluginRegistry struct {
	Plugins []PluginEntry `json:"plugins"`
}

// LoadRegistry reads the plugin registry from disk.
// Returns an empty registry if the file does not exist.
// Returns an error if the file exists but contains invalid JSON.
func LoadRegistry(path string) (*PluginRegistry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &PluginRegistry{}, nil
		}
		return nil, err
	}
	var reg PluginRegistry
	if err = json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("corrupt registry %s: %w", path, err)
	}
	return &reg, nil
}

// Save writes the registry to disk as indented JSON with 0644 permissions.
func (r *PluginRegistry) Save(path string) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Add appends a plugin entry to the registry.
// Returns an error if a plugin with the same name already exists.
func (r *PluginRegistry) Add(entry PluginEntry) error {
	if r.Find(entry.Name) != nil {
		return fmt.Errorf("plugin %q already registered", entry.Name)
	}
	r.Plugins = append(r.Plugins, entry)
	return nil
}

// Remove removes a plugin by name and returns the removed entry.
// Returns an error if the plugin is not found.
func (r *PluginRegistry) Remove(name string) (*PluginEntry, error) {
	for i, p := range r.Plugins {
		if p.Name == name {
			removed := r.Plugins[i]
			r.Plugins = append(r.Plugins[:i], r.Plugins[i+1:]...)
			return &removed, nil
		}
	}
	return nil, fmt.Errorf("plugin %q not found", name)
}

// Find returns a pointer to the entry with the given name, or nil if not found.
func (r *PluginRegistry) Find(name string) *PluginEntry {
	for i := range r.Plugins {
		if r.Plugins[i].Name == name {
			return &r.Plugins[i]
		}
	}
	return nil
}

// UpdateVersion updates the version and binary path for a named plugin.
// Returns an error if the plugin is not found.
func (r *PluginRegistry) UpdateVersion(name, version, binaryPath string) error {
	entry := r.Find(name)
	if entry == nil {
		return fmt.Errorf("plugin %q not found", name)
	}
	entry.Version = version
	entry.BinaryPath = binaryPath
	return nil
}

// Validate returns warnings for entries whose binary path does not exist on disk.
func (r *PluginRegistry) Validate() []string {
	var warnings []string
	for _, p := range r.Plugins {
		if _, err := os.Stat(p.BinaryPath); errors.Is(err, os.ErrNotExist) {
			warnings = append(warnings, fmt.Sprintf("plugin %q: binary not found at %s", p.Name, p.BinaryPath))
		}
	}
	return warnings
}
