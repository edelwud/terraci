package gitlab

import (
	"fmt"

	gitlabci "github.com/edelwud/terraci/plugins/gitlab/internal"
)

// ConfigKey returns the config key for the GitLab plugin.
func (p *Plugin) ConfigKey() string { return pluginName }

// NewConfig returns a new default GitLab config.
func (p *Plugin) NewConfig() any {
	return &gitlabci.Config{
		TerraformBinary: "terraform",
		Image:           gitlabci.Image{Name: "hashicorp/terraform:1.6"},
		StagesPrefix:    "deploy",
		Parallelism:     5,
		PlanEnabled:     true,
		InitEnabled:     true,
	}
}

// SetConfig sets the GitLab plugin configuration.
func (p *Plugin) SetConfig(cfg any) error {
	gc, ok := cfg.(*gitlabci.Config)
	if !ok {
		return fmt.Errorf("expected *Config, got %T", cfg)
	}
	p.cfg = gc
	p.configured = true
	return nil
}

// IsConfigured returns whether the plugin has been configured.
func (p *Plugin) IsConfigured() bool { return p.configured }
