package summaryengine

// Config holds summary plugin settings.
type Config struct {
	Enabled        *bool    `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	OnChangesOnly  bool     `yaml:"on_changes_only,omitempty" json:"on_changes_only,omitempty"`
	IncludeDetails bool     `yaml:"include_details,omitempty" json:"include_details,omitempty"`
	Labels         []string `yaml:"labels,omitempty" json:"labels,omitempty"`
}
