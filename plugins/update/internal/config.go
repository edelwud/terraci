// Package updateengine provides the core logic for Terraform dependency version checking and updating.
package updateengine

import (
	"fmt"
	"slices"
)

// Target constants define what dependencies to check.
const (
	TargetAll       = "all"
	TargetModules   = "modules"
	TargetProviders = "providers"
)

// Bump level constants.
const (
	BumpPatch = "patch"
	BumpMinor = "minor"
	BumpMajor = "major"
)

// UpdateConfig defines configuration for the update plugin.
type UpdateConfig struct {
	Enabled  bool     `yaml:"enabled" json:"enabled" jsonschema:"description=Enable dependency update checks,default=false"`
	Target   string   `yaml:"target,omitempty" json:"target,omitempty" jsonschema:"description=What to check: modules providers or all,default=all,enum=modules,enum=providers,enum=all"`
	Bump     string   `yaml:"bump,omitempty" json:"bump,omitempty" jsonschema:"description=Version bump level: patch minor or major,default=minor,enum=patch,enum=minor,enum=major"`
	Ignore   []string `yaml:"ignore,omitempty" json:"ignore,omitempty" jsonschema:"description=Provider or module sources to ignore"`
	Pipeline bool     `yaml:"pipeline,omitempty" json:"pipeline,omitempty" jsonschema:"description=Add dependency update check job to CI pipeline,default=false"`
}

// Validate checks if the config values are valid.
func (c *UpdateConfig) Validate() error {
	switch c.Target {
	case "", TargetAll, TargetModules, TargetProviders:
	default:
		return fmt.Errorf("invalid target %q: must be one of: all, modules, providers", c.Target)
	}
	switch c.Bump {
	case "", BumpPatch, BumpMinor, BumpMajor:
	default:
		return fmt.Errorf("invalid bump %q: must be one of: patch, minor, major", c.Bump)
	}
	return nil
}

// ShouldCheckProviders returns true if providers should be checked.
func (c *UpdateConfig) ShouldCheckProviders() bool {
	return c.Target == TargetAll || c.Target == TargetProviders || c.Target == ""
}

// ShouldCheckModules returns true if modules should be checked.
func (c *UpdateConfig) ShouldCheckModules() bool {
	return c.Target == TargetAll || c.Target == TargetModules || c.Target == ""
}

// IsIgnored returns true if the given source should be ignored.
func (c *UpdateConfig) IsIgnored(source string) bool {
	return slices.Contains(c.Ignore, source)
}
