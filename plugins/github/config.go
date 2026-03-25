package github

import (
	"fmt"

	githubci "github.com/edelwud/terraci/plugins/github/internal"
)

// ConfigKey returns the config key for the GitHub plugin.
func (p *Plugin) ConfigKey() string { return pluginName }

// NewConfig returns a new default GitHub config.
func (p *Plugin) NewConfig() any {
	return &githubci.Config{
		TerraformBinary: "terraform",
		RunsOn:          "ubuntu-latest",
		PlanEnabled:     true,
		InitEnabled:     true,
	}
}

// SetConfig sets the GitHub plugin configuration.
func (p *Plugin) SetConfig(cfg any) error {
	gc, ok := cfg.(*githubci.Config)
	if !ok {
		return fmt.Errorf("expected *Config, got %T", cfg)
	}
	p.cfg = gc
	p.configured = true
	return nil
}

// IsConfigured returns whether the plugin has been configured.
func (p *Plugin) IsConfigured() bool { return p.configured }
