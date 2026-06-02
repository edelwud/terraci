package plugin

import (
	"context"
	"fmt"

	"github.com/edelwud/terraci/pkg/config"
)

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

// ConfigLoader declares a config section under "extensions:" in .terraci.yaml.
// Implemented automatically by embedding BasePlugin[C] with a config type that
// implements Clone() C.
type ConfigLoader interface {
	Plugin
	ConfigKey() config.ExtensionKey
	SchemaConfig() any
	DecodeAndSet(config.ExtensionDocument) error
	IsConfigured() bool
	IsEnabled() bool
}

// ConfigError wraps config decode failures with stable plugin/key context.
type ConfigError struct {
	Plugin string
	Key    string
	Err    error
}

func (e ConfigError) Error() string {
	switch {
	case e.Plugin != "" && e.Key != "":
		return fmt.Sprintf("decode plugin config %s (%s): %v", e.Plugin, e.Key, e.Err)
	case e.Plugin != "":
		return fmt.Sprintf("decode plugin config %s: %v", e.Plugin, e.Err)
	default:
		return fmt.Sprintf("decode plugin config: %v", e.Err)
	}
}

func (e ConfigError) Unwrap() error {
	return e.Err
}
