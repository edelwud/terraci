package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
)

// CostConfig defines configuration for cost estimation.
type CostConfig struct {
	Enabled       bool                `yaml:"-" json:"-"`
	CacheDir      string              `yaml:"cache_dir,omitempty" json:"cache_dir,omitempty" jsonschema:"description=Directory to cache AWS pricing data"`
	CacheTTL      string              `yaml:"cache_ttl,omitempty" json:"cache_ttl,omitempty" jsonschema:"description=How long cached pricing is valid (e.g. 24h),default=24h"`
	Providers     CostProvidersConfig `yaml:"providers" json:"providers"`
	LegacyEnabled *bool               `yaml:"enabled,omitempty" json:"-"`
}

// CostProvidersConfig contains built-in provider configs.
type CostProvidersConfig struct {
	AWS *ProviderConfig `yaml:"aws,omitempty" json:"aws,omitempty"`
}

// ProviderConfig contains provider activation state.
type ProviderConfig struct {
	Enabled bool `yaml:"enabled,omitempty" json:"enabled,omitempty" jsonschema:"description=Enable this cloud provider,default=false"`
}

// EnabledProviderIDs returns all enabled cloud providers.
func (c *CostConfig) EnabledProviderIDs() []string {
	if c == nil {
		return nil
	}

	var providers []string
	if c.Providers.AWS != nil && c.Providers.AWS.Enabled {
		providers = append(providers, awskit.ProviderID)
	}
	if c.Enabled {
		providers = append(providers, awskit.ProviderID)
	}

	return providers
}

// HasEnabledProviders returns true when at least one provider is enabled.
func (c *CostConfig) HasEnabledProviders() bool {
	return len(c.EnabledProviderIDs()) > 0
}

// Validate checks if the CostConfig values are valid.
func (c *CostConfig) Validate() error {
	if c.LegacyEnabled != nil {
		return errors.New("plugins.cost.enabled is no longer supported; use plugins.cost.providers.aws.enabled")
	}
	if c.CacheTTL != "" {
		if _, err := time.ParseDuration(c.CacheTTL); err != nil {
			return fmt.Errorf("invalid cache_ttl %q: %w", c.CacheTTL, err)
		}
	}
	if !c.HasEnabledProviders() {
		return nil
	}
	return nil
}
