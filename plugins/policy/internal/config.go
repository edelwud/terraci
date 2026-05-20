package policyengine

import (
	"errors"
	"fmt"

	"go.yaml.in/yaml/v4"

	"github.com/edelwud/terraci/pkg/config/overwrite"
)

const DefaultNamespace = "terraform"

type SourceType string

const (
	SourceTypePath SourceType = "path"
	SourceTypeGit  SourceType = "git"
	SourceTypeOCI  SourceType = "oci"
)

// Config defines configuration for OPA policy checks.
type Config struct {
	Enabled        bool           `yaml:"enabled" json:"enabled" jsonschema:"description=Enable policy checks,default=false"`
	Sources        []SourceConfig `yaml:"sources,omitempty" json:"sources,omitempty" jsonschema:"description=Policy sources"`
	Namespaces     []string       `yaml:"namespaces,omitempty" json:"namespaces,omitempty" jsonschema:"description=Rego package namespaces to evaluate"`
	Decisions      Decisions      `yaml:"decisions" json:"decisions" jsonschema:"description=Actions for OPA decisions"`
	Overrides      []Override     `yaml:"overrides,omitempty" json:"overrides,omitempty" jsonschema:"description=Per-module policy configuration overrides"`
	SourceCacheDir string         `yaml:"source_cache_dir,omitempty" json:"source_cache_dir,omitempty" jsonschema:"description=Directory to cache materialized policies,default=.terraci/policies"`
}

// SourceConfig defines one policy source.
type SourceConfig struct {
	Type SourceType `yaml:"type" json:"type" jsonschema:"description=Policy source type,enum=path,enum=git,enum=oci,required"`
	Path string     `yaml:"path,omitempty" json:"path,omitempty" jsonschema:"description=Local policy directory path"`
	URL  string     `yaml:"url,omitempty" json:"url,omitempty" jsonschema:"description=Git repository or OCI artifact URL"`
	Ref  string     `yaml:"ref,omitempty" json:"ref,omitempty" jsonschema:"description=Git branch\\, tag\\, or commit SHA"`
}

// Override defines per-module policy overrides.
type Override struct {
	Match      string    `yaml:"match" json:"match" jsonschema:"description=Glob pattern to match module paths,required"`
	Enabled    *bool     `yaml:"enabled,omitempty" json:"enabled,omitempty" jsonschema:"description=Override enabled state for matching modules"`
	Namespaces []string  `yaml:"namespaces,omitempty" json:"namespaces,omitempty" jsonschema:"description=Override namespaces for matching modules"`
	Decisions  Decisions `yaml:"decisions" json:"decisions" jsonschema:"description=Override OPA decision actions"`
}

// Clone returns a deep copy of the policy configuration.
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}
	out := *c
	out.Sources = append([]SourceConfig(nil), c.Sources...)
	out.Namespaces = append([]string(nil), c.Namespaces...)
	out.Overrides = cloneOverrides(c.Overrides)
	return &out
}

func cloneOverrides(overrides []Override) []Override {
	if len(overrides) == 0 {
		return nil
	}
	out := make([]Override, len(overrides))
	for i, override := range overrides {
		out[i] = override
		if override.Enabled != nil {
			enabled := *override.Enabled
			out[i].Enabled = &enabled
		}
		out[i].Namespaces = append([]string(nil), override.Namespaces...)
	}
	return out
}

func (c *Config) UnmarshalYAML(unmarshal func(any) error) error {
	if err := rejectKeys(unmarshal, map[string]string{
		"failure_action": "decisions.deny",
		"warning_action": "decisions.warn",
		"cache_dir":      "source_cache_dir",
		"on_failure":     "decisions.deny",
		"on_warning":     "decisions.warn",
		"overwrites":     "overrides",
	}); err != nil {
		return err
	}

	type raw Config
	var decoded raw
	if err := unmarshal(&decoded); err != nil {
		return err
	}
	*c = Config(decoded)
	return nil
}

func (s *SourceConfig) UnmarshalYAML(unmarshal func(any) error) error {
	if err := rejectKeys(unmarshal, map[string]string{
		"git": "type: git + url",
		"oci": "type: oci + url",
	}); err != nil {
		return err
	}

	type raw SourceConfig
	var decoded raw
	if err := unmarshal(&decoded); err != nil {
		return err
	}
	*s = SourceConfig(decoded)
	return nil
}

func rejectKeys(unmarshal func(any) error, replacements map[string]string) error {
	var fields map[string]yaml.Node
	if err := unmarshal(&fields); err != nil {
		return err
	}
	for key := range fields {
		if replacement, ok := replacements[key]; ok {
			return fmt.Errorf("policy field %q is no longer supported; use %q", key, replacement)
		}
	}
	return nil
}

func (c *Config) Validate() error {
	if c == nil {
		return errors.New("policy config is nil")
	}
	if len(c.Sources) == 0 {
		return errors.New("at least one source is required when policy is enabled")
	}
	if err := c.Decisions.Validate(); err != nil {
		return err
	}

	for i := range c.Sources {
		if err := c.Sources[i].Validate(); err != nil {
			return fmt.Errorf("sources[%d]: %w", i, err)
		}
	}
	for i := range c.Overrides {
		if err := c.Overrides[i].Validate(); err != nil {
			return fmt.Errorf("overrides[%d]: %w", i, err)
		}
	}

	return nil
}

func (s *SourceConfig) Validate() error {
	if s == nil {
		return errors.New("source config is nil")
	}
	switch s.Type {
	case SourceTypePath:
		if s.Path == "" {
			return errors.New("path is required for path sources")
		}
		if s.URL != "" {
			return errors.New("url is not valid for path sources")
		}
		if s.Ref != "" {
			return errors.New("ref is not valid for path sources")
		}
	case SourceTypeGit:
		if s.URL == "" {
			return errors.New("url is required for git sources")
		}
		if s.Path != "" {
			return errors.New("path is not valid for git sources")
		}
	case SourceTypeOCI:
		if s.URL == "" {
			return errors.New("url is required for oci sources")
		}
		if s.Path != "" {
			return errors.New("path is not valid for oci sources")
		}
		if s.Ref != "" {
			return errors.New("ref is not valid for oci sources")
		}
	default:
		return errors.New("type must be one of: path, git, oci")
	}
	return nil
}

func (o Override) Validate() error {
	if o.Match == "" {
		return errors.New("match is required")
	}
	if err := overwrite.ValidatePathGlob(o.Match); err != nil {
		return fmt.Errorf("match: %w", err)
	}
	return o.Decisions.Validate()
}

func (c *Config) EffectiveConfig(modulePath string) (*Config, error) {
	if c == nil {
		return nil, nil
	}

	effective := c.Normalized()
	effective.Overrides = nil
	if err := overwrite.ApplyMatching(
		&effective,
		modulePath,
		c.Overrides,
		overwrite.ByPathGlob(func(ow *Override) string { return ow.Match }),
		applyOverride,
	); err != nil {
		return nil, err
	}
	return &effective, nil
}

func (c *Config) NamespacesOrDefault() []string {
	if c != nil && len(c.Namespaces) > 0 {
		return append([]string(nil), c.Namespaces...)
	}
	return []string{DefaultNamespace}
}

func (c *Config) CanBlock() bool {
	if c == nil {
		return false
	}
	return c.Decisions.CanBlock()
}

func (c *Config) Normalized() Config {
	if c == nil {
		return Config{Decisions: DefaultDecisions()}
	}
	out := *c
	out.Sources = append([]SourceConfig(nil), c.Sources...)
	out.Namespaces = append([]string(nil), c.Namespaces...)
	out.Overrides = append([]Override(nil), c.Overrides...)
	out.Decisions = c.Decisions.Normalize()
	return out
}

func applyOverride(effective *Config, override *Override) {
	if override.Enabled != nil {
		effective.Enabled = *override.Enabled
	}
	if len(override.Namespaces) > 0 {
		effective.Namespaces = append([]string(nil), override.Namespaces...)
	}
	if override.Decisions.Deny != "" {
		effective.Decisions.Deny = override.Decisions.Deny
	}
	if override.Decisions.Warn != "" {
		effective.Decisions.Warn = override.Decisions.Warn
	}
}
