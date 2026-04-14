// Package tfupdateengine provides the core logic for Terraform dependency resolution and lock synchronization.
package tfupdateengine

import (
	"errors"
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

const (
	DefaultRegistryHost           = "registry.terraform.io"
	DefaultMetadataCacheBackend   = "inmemcache"
	DefaultMetadataCacheNamespace = "tfupdate/registry"
	DefaultArtifactCacheBackend   = "diskblob"
	DefaultArtifactCacheNamespace = "tfupdate/providers"
	DefaultMetadataCacheTTL       = 6 * time.Hour
	DefaultReadTimeout            = 5 * time.Minute
	DefaultWriteTimeout           = 20 * time.Minute

	DefaultCacheBackend   = DefaultMetadataCacheBackend
	DefaultCacheNamespace = DefaultMetadataCacheNamespace
	DefaultCacheTTL       = DefaultMetadataCacheTTL
)

// UpdateConfig defines configuration for the tfupdate plugin.
type UpdateConfig struct {
	Enabled    bool           `yaml:"enabled" json:"enabled" jsonschema:"description=Enable terraform dependency resolution,default=false"`
	Target     string         `yaml:"target,omitempty" json:"target,omitempty" jsonschema:"description=What to inspect: modules providers or all,default=all,enum=modules,enum=providers,enum=all"`
	Ignore     []string       `yaml:"ignore,omitempty" json:"ignore,omitempty" jsonschema:"description=Provider or module sources to ignore"`
	Pipeline   bool           `yaml:"pipeline,omitempty" json:"pipeline,omitempty" jsonschema:"description=Add tfupdate check job to CI pipeline,default=false"`
	Timeout    string         `yaml:"timeout,omitempty" json:"timeout,omitempty" jsonschema:"description=Overall timeout for a tfupdate run (e.g. 15m)"`
	Policy     UpdatePolicy   `yaml:"policy,omitempty" json:"policy,omitempty"`         //nolint:modernize // omitzero unsupported by yaml/v4
	Registries RegistryConfig `yaml:"registries,omitempty" json:"registries,omitempty"` //nolint:modernize // omitzero unsupported by yaml/v4
	Lock       LockConfig     `yaml:"lock,omitempty" json:"lock,omitempty"`             //nolint:modernize // omitzero unsupported by yaml/v4
	Cache      *CacheConfig   `yaml:"cache,omitempty" json:"cache,omitempty"`

	Bump                string   `yaml:"-" json:"-"`
	Pin                 bool     `yaml:"-" json:"-"`
	LegacyLockPlatforms []string `yaml:"-" json:"-"`
}

type UpdatePolicy struct {
	Bump string `yaml:"bump,omitempty" json:"bump,omitempty" jsonschema:"description=Version bump level: patch minor or major,enum=patch,enum=minor,enum=major"`
	Pin  bool   `yaml:"pin,omitempty" json:"pin,omitempty" jsonschema:"description=Pin applied dependency constraints to an exact version,default=false"`
}

type RegistryConfig struct {
	Default   string            `yaml:"default,omitempty" json:"default,omitempty" jsonschema:"description=Default registry hostname for modules/providers without lock-based host information,default=registry.terraform.io"`
	Providers map[string]string `yaml:"providers,omitempty" json:"providers,omitempty" jsonschema:"description=Registry hostname overrides keyed by normalized short provider source, e.g. hashicorp/aws"`
}

type LockConfig struct {
	Platforms []string `yaml:"platforms,omitempty" json:"platforms,omitempty" jsonschema:"description=Required platform set for provider h1 hashes, e.g. linux_amd64 darwin_arm64"`
}

type CacheConfig struct {
	Backend   string              `yaml:"backend,omitempty" json:"backend,omitempty" jsonschema:"description=Legacy metadata cache backend plugin name"`
	TTL       string              `yaml:"ttl,omitempty" json:"ttl,omitempty" jsonschema:"description=Legacy metadata cache ttl"`
	Namespace string              `yaml:"namespace,omitempty" json:"namespace,omitempty" jsonschema:"description=Legacy metadata cache namespace"`
	Metadata  MetadataCacheConfig `yaml:"metadata,omitempty" json:"metadata,omitempty"`   //nolint:modernize // omitzero unsupported by yaml/v4
	Artifacts ArtifactCacheConfig `yaml:"artifacts,omitempty" json:"artifacts,omitempty"` //nolint:modernize // omitzero unsupported by yaml/v4
}

type MetadataCacheConfig struct {
	Backend   string `yaml:"backend,omitempty" json:"backend,omitempty" jsonschema:"description=KV cache backend plugin name,default=inmemcache"`
	TTL       string `yaml:"ttl,omitempty" json:"ttl,omitempty" jsonschema:"description=How long registry metadata stays cached,default=6h"`
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty" jsonschema:"description=Namespace for tfupdate registry metadata cache,default=tfupdate/registry"`
}

type ArtifactCacheConfig struct {
	Backend   string `yaml:"backend,omitempty" json:"backend,omitempty" jsonschema:"description=Blob store backend plugin name,default=diskblob"`
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty" jsonschema:"description=Namespace for cached provider archives and hashes,default=tfupdate/providers"`
}

func DefaultConfig() *UpdateConfig {
	return &UpdateConfig{
		Target: TargetAll,
		Registries: RegistryConfig{
			Default: DefaultRegistryHost,
		},
		Cache: &CacheConfig{
			Metadata: MetadataCacheConfig{
				Backend:   DefaultMetadataCacheBackend,
				Namespace: DefaultMetadataCacheNamespace,
				TTL:       DefaultMetadataCacheTTL.String(),
			},
			Artifacts: ArtifactCacheConfig{
				Backend:   DefaultArtifactCacheBackend,
				Namespace: DefaultArtifactCacheNamespace,
			},
		},
	}
}

// Validate checks static config values that are safe to validate before CLI overrides are applied.
func (c *UpdateConfig) Validate() error {
	switch c.Target {
	case "", TargetAll, TargetModules, TargetProviders:
	default:
		return fmt.Errorf("invalid target %q: must be one of: all, modules, providers", c.Target)
	}

	if c.BumpPolicy() != "" {
		switch c.BumpPolicy() {
		case BumpPatch, BumpMinor, BumpMajor:
		default:
			return fmt.Errorf("invalid policy.bump %q: must be one of: patch, minor, major", c.BumpPolicy())
		}
	}

	if c.CacheMetadataTTLRaw() != "" {
		if _, err := time.ParseDuration(c.CacheMetadataTTLRaw()); err != nil {
			return fmt.Errorf("invalid cache.metadata.ttl %q: %w", c.CacheMetadataTTLRaw(), err)
		}
	}
	if c.Timeout != "" {
		if _, err := time.ParseDuration(c.Timeout); err != nil {
			return fmt.Errorf("invalid timeout %q: %w", c.Timeout, err)
		}
	}

	for source, host := range c.Registries.Providers {
		if source == "" {
			return errors.New("registries.providers contains an empty source key")
		}
		if host == "" {
			return fmt.Errorf("registries.providers[%q] must not be empty", source)
		}
	}

	return nil
}

// ValidateRuntime checks values after CLI overrides have been applied.
func (c *UpdateConfig) ValidateRuntime() error {
	if err := c.Validate(); err != nil {
		return err
	}
	if c.BumpPolicy() == "" {
		return errors.New("tfupdate policy.bump is required (set plugins.tfupdate.policy.bump or pass --bump)")
	}
	return nil
}

func (c *UpdateConfig) ShouldCheckProviders() bool {
	return c.Target == TargetAll || c.Target == TargetProviders || c.Target == ""
}

func (c *UpdateConfig) ShouldCheckModules() bool {
	return c.Target == TargetAll || c.Target == TargetModules || c.Target == ""
}

func (c *UpdateConfig) IsIgnored(source string) bool {
	return slices.Contains(c.Ignore, source)
}

func (c *UpdateConfig) BumpPolicy() string {
	if c == nil {
		return ""
	}
	if c.Policy.Bump != "" {
		return c.Policy.Bump
	}
	if c.Bump != "" {
		return c.Bump
	}
	return c.Policy.Bump
}

func (c *UpdateConfig) PinEnabled() bool {
	return c != nil && (c.Policy.Pin || c.Pin)
}

func (c *UpdateConfig) ProviderRegistryHost(source string) string {
	if c != nil && c.Registries.Providers != nil {
		if host := c.Registries.Providers[source]; host != "" {
			return host
		}
	}
	return c.DefaultRegistryHost()
}

func (c *UpdateConfig) DefaultRegistryHost() string {
	if c == nil || c.Registries.Default == "" {
		return DefaultRegistryHost
	}
	return c.Registries.Default
}

func (c *UpdateConfig) LockPlatforms() []string {
	if c == nil {
		return nil
	}
	if len(c.Lock.Platforms) > 0 {
		return slices.Clone(c.Lock.Platforms)
	}
	if len(c.LegacyLockPlatforms) > 0 {
		return slices.Clone(c.LegacyLockPlatforms)
	}
	return nil
}

func (c *UpdateConfig) MetadataCacheBackend() string {
	if c == nil || c.Cache == nil {
		return DefaultMetadataCacheBackend
	}
	if c.Cache.Metadata.Backend != "" {
		return c.Cache.Metadata.Backend
	}
	if c.Cache.Backend != "" {
		return c.Cache.Backend
	}
	return DefaultMetadataCacheBackend
}

func (c *UpdateConfig) MetadataCacheNamespace() string {
	if c == nil || c.Cache == nil {
		return DefaultMetadataCacheNamespace
	}
	if c.Cache.Metadata.Namespace != "" {
		return c.Cache.Metadata.Namespace
	}
	if c.Cache.Namespace != "" {
		return c.Cache.Namespace
	}
	return DefaultMetadataCacheNamespace
}

func (c *UpdateConfig) MetadataCacheTTL() time.Duration {
	if c == nil || c.CacheMetadataTTLRaw() == "" {
		return DefaultMetadataCacheTTL
	}
	ttl, err := time.ParseDuration(c.CacheMetadataTTLRaw())
	if err != nil {
		return DefaultMetadataCacheTTL
	}
	return ttl
}

func (c *UpdateConfig) ArtifactCacheBackend() string {
	if c == nil || c.Cache == nil || c.Cache.Artifacts.Backend == "" {
		return DefaultArtifactCacheBackend
	}
	return c.Cache.Artifacts.Backend
}

func (c *UpdateConfig) ArtifactCacheNamespace() string {
	if c == nil || c.Cache == nil || c.Cache.Artifacts.Namespace == "" {
		return DefaultArtifactCacheNamespace
	}
	return c.Cache.Artifacts.Namespace
}

func (c *UpdateConfig) CacheMetadataTTLRaw() string {
	if c == nil || c.Cache == nil {
		return ""
	}
	if c.Cache.Metadata.TTL != "" {
		return c.Cache.Metadata.TTL
	}
	return c.Cache.TTL
}

func (c *UpdateConfig) CacheBackend() string    { return c.MetadataCacheBackend() }
func (c *UpdateConfig) CacheNamespace() string  { return c.MetadataCacheNamespace() }
func (c *UpdateConfig) CacheTTL() time.Duration { return c.MetadataCacheTTL() }

func (c *UpdateConfig) CommandTimeout(write bool) time.Duration {
	if c != nil && c.Timeout != "" {
		if timeout, err := time.ParseDuration(c.Timeout); err == nil {
			return timeout
		}
	}
	if write {
		return DefaultWriteTimeout
	}
	return DefaultReadTimeout
}
