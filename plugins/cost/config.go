package cost

import (
	"fmt"

	"github.com/edelwud/terraci/pkg/plugin"
	costengine "github.com/edelwud/terraci/plugins/cost/internal"
)

// ConfigKey returns the config key for the cost plugin.
func (p *Plugin) ConfigKey() string { return "cost" }

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

// IsConfigured returns whether the plugin has been configured.
func (p *Plugin) IsConfigured() bool { return p.configured }

func (p *Plugin) effectiveConfig(_ *plugin.AppContext) *costengine.CostConfig {
	if p.cfg != nil {
		return p.cfg
	}
	return &costengine.CostConfig{}
}

func (p *Plugin) getEstimator(cfg *costengine.CostConfig) *costengine.Estimator {
	if p.estimator != nil {
		return p.estimator
	}
	// Fallback: create on-demand if Initialize was not called (e.g., version/schema commands)
	return costengine.NewEstimatorFromConfig(cfg)
}
