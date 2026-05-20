package plugin

import "fmt"

// Validator is implemented by plugins that want the registry to perform a
// startup sanity check. The framework calls Validate() once after the plugin
// is constructed by its factory; a non-nil error panics in RegisterFactory
// with a clear message identifying the misconfigured plugin.
type Validator interface {
	Validate() error
}

// ConfigCloner is the config contract required by BasePlugin. Clone must
// return a deep copy of the concrete plugin config. Pointer config types should
// handle a nil receiver and return nil.
type ConfigCloner[C any] interface {
	Clone() C
}

// BasePlugin provides shared implementation for all plugins that have configuration.
// C is the plugin's concrete config type. Embedding this gives you:
//   - Name(), Description()
//   - ConfigKey(), NewConfig(), DecodeAndSet(), IsConfigured(), IsEnabled()
//   - Config() (typed defensive-copy access to config)
//   - Reset() (resets config state; override to reset custom fields)
//   - Validate() (registration-time sanity check; see Validator)
type BasePlugin[C ConfigCloner[C]] struct {
	PluginName string
	PluginDesc string
	PluginKey  string       // config key; defaults to PluginName if empty
	EnableMode EnablePolicy // how the framework checks if this plugin is active
	DefaultCfg func() C     // factory for default config

	// IsEnabledFn is an optional custom check for EnabledExplicitly and EnabledByDefault.
	// For EnabledExplicitly: called when configured, must return true to activate.
	// For EnabledByDefault: called when configured, return false to deactivate.
	IsEnabledFn func(C) bool

	cfg        C
	configured bool
}

// Name returns the plugin's unique identifier.
func (b *BasePlugin[C]) Name() string { return b.PluginName }

// Description returns a human-readable description.
func (b *BasePlugin[C]) Description() string { return b.PluginDesc }

// ConfigKey returns the config section key under "extensions:" in .terraci.yaml.
func (b *BasePlugin[C]) ConfigKey() string {
	if b.PluginKey != "" {
		return b.PluginKey
	}
	return b.PluginName
}

// NewConfig returns a new instance of the default config for schema generation.
func (b *BasePlugin[C]) NewConfig() any {
	return b.DefaultCfg().Clone()
}

// DecodeAndSet decodes plugin config via the provided decoder and stores it.
func (b *BasePlugin[C]) DecodeAndSet(decode func(target any) error) error {
	cfg := b.DefaultCfg()
	if err := decode(&cfg); err != nil {
		return err
	}
	b.cfg = cfg.Clone()
	b.configured = true
	return nil
}

// Config returns a defensive copy of the typed plugin configuration. Mutating
// the returned value never changes plugin state.
func (b *BasePlugin[C]) Config() C { return b.cfg.Clone() }

// SetTypedConfig sets the typed config directly (used by tests and flag overrides).
func (b *BasePlugin[C]) SetTypedConfig(cfg C) {
	b.cfg = cfg.Clone()
	b.configured = true
}

// IsConfigured returns true if config was loaded for this plugin.
func (b *BasePlugin[C]) IsConfigured() bool { return b.configured }

// IsEnabled returns whether the plugin should be active, based on EnablePolicy.
func (b *BasePlugin[C]) IsEnabled() bool {
	switch b.EnableMode {
	case EnabledWhenConfigured:
		return b.configured
	case EnabledExplicitly:
		if !b.configured {
			return false
		}
		if b.IsEnabledFn != nil {
			return b.IsEnabledFn(b.cfg)
		}
		return false
	case EnabledByDefault:
		if !b.configured {
			return true
		}
		if b.IsEnabledFn != nil {
			return b.IsEnabledFn(b.cfg)
		}
		return true
	case EnabledAlways:
		return true
	}
	return false
}

// Reset resets the config state. Override in your plugin to also reset custom fields.
func (b *BasePlugin[C]) Reset() {
	var zero C
	b.cfg = zero
	b.configured = false
}

// Validate performs registration-time sanity checks on the BasePlugin
// embedding. It is invoked by registry.RegisterFactory; a non-nil error
// panics there with a message identifying the misconfigured plugin.
//
// Currently catches the most common silent-disable bug: a plugin that opts
// into EnabledExplicitly but forgets to set IsEnabledFn — IsEnabled() would
// always return false, so the plugin appears registered but never runs.
func (b *BasePlugin[C]) Validate() error {
	if b.EnableMode == EnabledExplicitly && b.IsEnabledFn == nil {
		return fmt.Errorf(
			"plugin %q uses EnabledExplicitly without IsEnabledFn — IsEnabled() would always return false",
			b.PluginName,
		)
	}
	return nil
}
