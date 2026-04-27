package plugin

import "context"

// --- Lifecycle interfaces ---
//
// Preferred plugin flow:
//   - register via init() + registry.RegisterFactory()
//   - configure via ConfigLoader
//   - preflight via Preflightable
//   - build lazy command-time runtime via RuntimeProvider
//   - execute plugin-local use-cases and rendering
//
// IsConfigured() answers whether config exists; IsEnabled() answers whether the
// plugin should actively participate in the current run. Framework-owned
// lifecycle should key off IsEnabled().

// Preflightable plugins run cheap validation after config is loaded, before any
// command runs. Preflight should stay side-effect-light: do not cache mutable
// command state or perform heavy runtime setup that can be created lazily
// inside plugin use-cases.
type Preflightable interface {
	Plugin
	Preflight(ctx context.Context, appCtx *AppContext) error
}

// --- Configuration ---

// ConfigLoader declares a config section under "plugins:" in .terraci.yaml.
// Implemented automatically by embedding BasePlugin[C].
type ConfigLoader interface {
	Plugin
	ConfigKey() string
	NewConfig() any
	DecodeAndSet(decode func(target any) error) error
	IsConfigured() bool
	IsEnabled() bool
}
