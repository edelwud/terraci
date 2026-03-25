package policy

import (
	"fmt"

	"github.com/edelwud/terraci/pkg/plugin"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

// ConfigKey returns the config key for the policy plugin.
func (p *Plugin) ConfigKey() string { return "policy" }

// NewConfig returns a new default policy config.
func (p *Plugin) NewConfig() any { return &policyengine.Config{} }

// SetConfig sets the policy plugin configuration.
func (p *Plugin) SetConfig(cfg any) error {
	pc, ok := cfg.(*policyengine.Config)
	if !ok {
		return fmt.Errorf("expected *Config, got %T", cfg)
	}
	p.cfg = pc
	p.configured = true
	return nil
}

// IsConfigured returns whether the plugin has been configured.
func (p *Plugin) IsConfigured() bool { return p.configured }

func (p *Plugin) effectiveConfig(_ *plugin.AppContext) *policyengine.Config {
	if p.cfg != nil {
		return p.cfg
	}
	return &policyengine.Config{}
}
