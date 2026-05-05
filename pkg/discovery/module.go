// Package discovery provides functionality for discovering Terraform modules
// in a directory structure following a configurable pattern like: {service}/{environment}/{region}/{module}
package discovery

import (
	"maps"
	"path/filepath"
)

// Module represents a discovered Terraform module with its path components.
type Module struct {
	components map[string]string
	segments   []string

	Path         string  // Full path to the module directory
	RelativePath string  // Relative path from the root directory
	Parent       *Module // Parent module reference (for submodules)
	Children     []*Module

	// IsLibrary marks the module as a non-executable shared/library module.
	// Set when the module's relative path lies under any configured
	// library_modules.paths root. Library modules are excluded from
	// executable target selection but tracked separately for reporting and
	// change-detection.
	IsLibrary bool
}

// NewModule creates a Module from ordered segment names and values.
func NewModule(segments, values []string, path, relPath string) *Module {
	components := make(map[string]string, len(segments))
	for i, seg := range segments {
		if i < len(values) {
			components[seg] = values[i]
		}
	}
	return &Module{
		components:   components,
		segments:     segments,
		Path:         path,
		RelativePath: relPath,
	}
}

// Get returns the value of a named component.
func (m *Module) Get(name string) string { return m.components[name] }

// SetComponent sets a component value.
func (m *Module) SetComponent(name, value string) { m.components[name] = value }

// Segments returns the ordered segment names.
func (m *Module) Segments() []string { return m.segments }

// Components returns a copy of the component map.
func (m *Module) Components() map[string]string {
	return maps.Clone(m.components)
}

// ID returns a unique identifier for the module (its relative path).
func (m *Module) ID() string { return m.RelativePath }

// String returns the module ID.
func (m *Module) String() string { return m.ID() }

// LeafValue returns the value of the last pattern segment.
func (m *Module) LeafValue() string {
	if len(m.segments) == 0 {
		return ""
	}
	return m.components[m.segments[len(m.segments)-1]]
}

// Name returns the leaf name including submodule if present (e.g. "eks" or "ec2/rabbitmq").
func (m *Module) Name() string {
	leaf := m.LeafValue()
	if sub := m.Get("submodule"); sub != "" {
		return leaf + "/" + sub
	}
	return leaf
}

// IsSubmodule returns true if this module is a submodule.
func (m *Module) IsSubmodule() bool { return m.Get("submodule") != "" }

// ContextPrefix returns the path prefix for context-relative lookups
// (all segment values except the last, joined with /).
func (m *Module) ContextPrefix() string {
	if len(m.segments) <= 1 {
		return ""
	}
	parts := make([]string, 0, len(m.segments)-1)
	for _, seg := range m.segments[:len(m.segments)-1] {
		parts = append(parts, m.components[seg])
	}
	return filepath.Join(parts...)
}
