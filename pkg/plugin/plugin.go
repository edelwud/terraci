// Package plugin provides the compile-time plugin system for TerraCi.
// Plugins register themselves via init() and blank imports, following the
// same pattern as database/sql drivers and Caddy modules.
//
// Core types (interfaces, BasePlugin, AppContext) live in this package.
// Plugin factories and per-command registries live in pkg/plugin/registry.
// Init wizard types live in pkg/plugin/initwiz.
package plugin

// Plugin is the core interface every plugin must implement.
type Plugin interface {
	// Name returns a unique identifier (e.g., "gitlab", "cost", "slack").
	Name() string
	// Description returns a human-readable description.
	Description() string
}
