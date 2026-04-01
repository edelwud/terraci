// Package updateengine provides the core logic for Terraform dependency version checking and updating.
package updateengine

import (
	"fmt"
	"slices"
	"time"
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
	Enabled  bool         `yaml:"enabled" json:"enabled" jsonschema:"description=Enable dependency update checks,default=false"`
	Target   string       `yaml:"target,omitempty" json:"target,omitempty" jsonschema:"description=What to check: modules providers or all,default=all,enum=modules,enum=providers,enum=all"`
	Bump     string       `yaml:"bump,omitempty" json:"bump,omitempty" jsonschema:"description=Version bump level: patch minor or major,default=minor,enum=patch,enum=minor,enum=major"`
	Ignore   []string     `yaml:"ignore,omitempty" json:"ignore,omitempty" jsonschema:"description=Provider or module sources to ignore"`
	Pipeline bool         `yaml:"pipeline,omitempty" json:"pipeline,omitempty" jsonschema:"description=Add dependency update check job to CI pipeline,default=false"`
	Cache    *CacheConfig `yaml:"cache,omitempty" json:"cache,omitempty" jsonschema:"description=Cache backend selection and policy for registry version lookups"`
}

// CacheConfig defines cache selection and policy for the update plugin.
type CacheConfig struct {
	Backend   string `yaml:"backend,omitempty" json:"backend,omitempty" jsonschema:"description=KV cache backend plugin name,default=inmemcache"`
	TTL       string `yaml:"ttl,omitempty" json:"ttl,omitempty" jsonschema:"description=How long cached registry lookups remain valid (e.g. 6h),default=6h"`
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty" jsonschema:"description=Cache namespace for update registry lookups,default=update/registry"`
}

const (
	DefaultCacheBackend   = "inmemcache"
	DefaultCacheNamespace = "update/registry"
	DefaultCacheTTL       = 6 * time.Hour
)

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
	if c.Cache != nil && c.Cache.TTL != "" {
		if _, err := time.ParseDuration(c.Cache.TTL); err != nil {
			return fmt.Errorf("invalid cache ttl %q: %w", c.Cache.TTL, err)
		}
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

// CacheBackend returns the configured cache backend or the built-in default.
func (c *UpdateConfig) CacheBackend() string {
	if c == nil || c.Cache == nil || c.Cache.Backend == "" {
		return DefaultCacheBackend
	}
	return c.Cache.Backend
}

// CacheNamespace returns the configured cache namespace or the built-in default.
func (c *UpdateConfig) CacheNamespace() string {
	if c == nil || c.Cache == nil || c.Cache.Namespace == "" {
		return DefaultCacheNamespace
	}
	return c.Cache.Namespace
}

// CacheTTL returns the configured cache TTL or the built-in default.
func (c *UpdateConfig) CacheTTL() time.Duration {
	if c == nil || c.Cache == nil || c.Cache.TTL == "" {
		return DefaultCacheTTL
	}
	ttl, err := time.ParseDuration(c.Cache.TTL)
	if err != nil {
		return DefaultCacheTTL
	}
	return ttl
}
