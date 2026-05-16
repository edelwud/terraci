package summaryengine

// Config holds summary plugin settings.
type Config struct {
	Enabled        *bool    `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	OnChangesOnly  bool     `yaml:"on_changes_only,omitempty" json:"on_changes_only,omitempty"`
	IncludeDetails *bool    `yaml:"include_details,omitempty" json:"include_details,omitempty"`
	Labels         []string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

// Normalized returns a value copy with stable defaults and owned slices.
func (c *Config) Normalized() Config {
	if c == nil {
		return Config{}
	}
	out := *c
	if c.Enabled != nil {
		enabled := *c.Enabled
		out.Enabled = &enabled
	}
	if c.IncludeDetails != nil {
		includeDetails := *c.IncludeDetails
		out.IncludeDetails = &includeDetails
	}
	if c.Labels != nil {
		out.Labels = append([]string(nil), c.Labels...)
	}
	return out
}

// IncludeDetailsEnabled reports whether detailed report sections should be rendered.
func (c *Config) IncludeDetailsEnabled() bool {
	return c.IncludeDetails == nil || *c.IncludeDetails
}
