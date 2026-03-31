package plugin

// EnablePolicy controls how the framework determines if a plugin is active.
type EnablePolicy int

const (
	// EnabledWhenConfigured means the plugin is active if its config section
	// exists in .terraci.yaml (e.g., gitlab, github).
	EnabledWhenConfigured EnablePolicy = iota

	// EnabledExplicitly requires an explicit opt-in beyond having config (e.g., cost, policy).
	// When IsEnabledFn is set: called after config is loaded, must return true to activate.
	// When IsEnabledFn is nil: always returns false, even if configured.
	EnabledExplicitly

	// EnabledByDefault means the plugin is active unless enabled: false is set (e.g., summary).
	EnabledByDefault

	// EnabledAlways means the plugin is always active regardless of config (e.g., git).
	EnabledAlways
)
