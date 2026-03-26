package summary

import "fmt"

// Config holds summary plugin settings.
type Config struct {
	Enabled        *bool    `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	OnChangesOnly  bool     `yaml:"on_changes_only,omitempty" json:"on_changes_only,omitempty"`
	IncludeDetails bool     `yaml:"include_details,omitempty" json:"include_details,omitempty"`
	Labels         []string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

func (p *Plugin) ConfigKey() string { return "summary" }
func (p *Plugin) NewConfig() any    { return &Config{} }

func (p *Plugin) SetConfig(cfg any) error {
	c, ok := cfg.(*Config)
	if !ok {
		return fmt.Errorf("summary: expected *Config, got %T", cfg)
	}
	p.cfg = c
	p.configured = true
	return nil
}

func (p *Plugin) IsConfigured() bool { return p.configured }

func (p *Plugin) isEnabled() bool {
	if p.cfg == nil || p.cfg.Enabled == nil {
		return true // default enabled
	}
	return *p.cfg.Enabled
}
