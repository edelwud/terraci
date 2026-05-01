package model

import (
	"errors"
	"fmt"
	"time"
)

type CostConfig struct {
	CacheDir      string              `yaml:"cache_dir,omitempty" json:"cache_dir,omitempty" jsonschema:"description=Deprecated and unsupported; use extensions.diskblob.root_dir instead"`
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

// CostProvidersConfig maps provider config-keys (e.g. "aws") to their enable flag.
// Keys must match the ConfigKey declared in the corresponding cloud.Definition.
type CostProvidersConfig map[string]ProviderConfig

// ProviderConfig contains provider activation state.
type ProviderConfig struct {
	Enabled bool `yaml:"enabled,omitempty" json:"enabled,omitempty" jsonschema:"description=Enable this cloud provider,default=false"`
}

// HasEnabledProviders returns true when at least one provider is enabled.
func (c *CostConfig) HasEnabledProviders() bool {
	if c == nil {
		return false
	}
	for _, pc := range c.Providers {
		if pc.Enabled {
			return true
		}
	}
	return false
}

// Validate checks if the CostConfig values are valid.
func (c *CostConfig) Validate() error {
	if c.LegacyEnabled != nil {
		return errors.New("extensions.cost.enabled is no longer supported; use extensions.cost.providers.aws.enabled")
	}
	if c.CacheDir != "" {
		return errors.New("extensions.cost.cache_dir is no longer supported; use extensions.diskblob.root_dir")
	}
	if c.BlobCache != nil && c.BlobCache.TTL != "" {
		if _, err := time.ParseDuration(c.BlobCache.TTL); err != nil {
			return fmt.Errorf("invalid blob_cache.ttl %q: %w", c.BlobCache.TTL, err)
		}
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

// blobCacheTTL returns the configured blob cache TTL string or empty when unset.
func (c *CostConfig) blobCacheTTL() string {
	if c == nil || c.BlobCache == nil {
		return ""
	}
	return c.BlobCache.TTL
}

// DefaultCacheTTL is how long pricing cache data is considered valid.
const DefaultCacheTTL = 24 * time.Hour

// CacheTTLDuration returns the configured blob cache TTL, or the default 24h when unset or invalid.
func (c *CostConfig) CacheTTLDuration() time.Duration {
	if c == nil {
		return DefaultCacheTTL
	}
	if ttl := c.blobCacheTTL(); ttl != "" {
		if d, err := time.ParseDuration(ttl); err == nil {
			return d
		}
	}
	return DefaultCacheTTL
}
