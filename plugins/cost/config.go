package cost

import (
	"fmt"

	costengine "github.com/edelwud/terraci/plugins/cost/internal"
)

// ConfigKey returns the config key for the cost plugin.
func (p *Plugin) ConfigKey() string { return pluginName }

// NewConfig returns a new default cost config.
func (p *Plugin) NewConfig() any { return &costengine.CostConfig{} }

// SetConfig sets the cost plugin configuration.
func (p *Plugin) SetConfig(cfg any) error {
	cc, ok := cfg.(*costengine.CostConfig)
	if !ok {
		return fmt.Errorf("expected *CostConfig, got %T", cfg)
	}
	p.cfg = cc
	p.configured = true
	return nil
}

// IsConfigured returns whether the plugin has been configured and enabled.
func (p *Plugin) IsConfigured() bool { return p.configured && p.cfg != nil && p.cfg.Enabled }

func (p *Plugin) getEstimator() *costengine.Estimator {
	return p.estimator
}
