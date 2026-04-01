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
	CacheDir      string              `yaml:"cache_dir,omitempty" json:"cache_dir,omitempty" jsonschema:"description=Deprecated and unsupported; use plugins.diskblob.root_dir instead"`
	BlobCache     *BlobCacheConfig    `yaml:"blob_cache,omitempty" json:"blob_cache,omitempty" jsonschema:"description=Blob cache backend selection for pricing data"`
	Providers     CostProvidersConfig `yaml:"providers" json:"providers"`
	LegacyEnabled *bool               `yaml:"enabled,omitempty" json:"-"`
}

// BlobCacheConfig selects a blob backend for pricing data.
type BlobCacheConfig struct {
	Backend   string `yaml:"backend,omitempty" json:"backend,omitempty" jsonschema:"description=Blob cache backend plugin name,default=diskblob"`
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty" jsonschema:"description=Blob cache namespace for pricing data,default=cost/pricing"`
	TTL       string `yaml:"ttl,omitempty" json:"ttl,omitempty" jsonschema:"description=How long cached pricing is valid (e.g. 24h),default=24h"`
}

const (
	DefaultBlobCacheBackend   = "diskblob"
	DefaultBlobCacheNamespace = "cost/pricing"
)

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
	if c.CacheDir != "" {
		return errors.New("plugins.cost.cache_dir is no longer supported; use plugins.diskblob.root_dir")
	}
	if c.BlobCache != nil && c.BlobCache.TTL != "" {
		if _, err := time.ParseDuration(c.BlobCache.TTL); err != nil {
			return fmt.Errorf("invalid blob_cache.ttl %q: %w", c.BlobCache.TTL, err)
		}
	}
	if !c.HasEnabledProviders() {
		return nil
	}
	return nil
}

// BlobCacheBackend returns the configured blob backend or the built-in default.
func (c *CostConfig) BlobCacheBackend() string {
	if c == nil || c.BlobCache == nil || c.BlobCache.Backend == "" {
		return DefaultBlobCacheBackend
	}
	return c.BlobCache.Backend
}

// BlobCacheNamespace returns the configured blob namespace or the built-in default.
func (c *CostConfig) BlobCacheNamespace() string {
	if c == nil || c.BlobCache == nil || c.BlobCache.Namespace == "" {
		return DefaultBlobCacheNamespace
	}
	return c.BlobCache.Namespace
}

// BlobCacheTTL returns the configured blob cache TTL string or empty when unset.
func (c *CostConfig) BlobCacheTTL() string {
	if c == nil || c.BlobCache == nil {
		return ""
	}
	return c.BlobCache.TTL
}
