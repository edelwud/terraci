package summary

import (
	"fmt"

	summaryengine "github.com/edelwud/terraci/plugins/summary/internal"
)

func (p *Plugin) ConfigKey() string { return pluginName }
func (p *Plugin) NewConfig() any    { return &summaryengine.Config{} }

func (p *Plugin) SetConfig(cfg any) error {
	c, ok := cfg.(*summaryengine.Config)
	if !ok {
		return fmt.Errorf("expected *Config, got %T", cfg)
	}
	p.cfg = c
	p.configured = true
	return nil
}

// IsConfigured returns whether the plugin has been configured and enabled.
func (p *Plugin) IsConfigured() bool {
	return p.configured && p.cfg != nil && (p.cfg.Enabled == nil || *p.cfg.Enabled)
}
