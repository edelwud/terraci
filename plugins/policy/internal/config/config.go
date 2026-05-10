package config

import (
	"errors"
	"fmt"

	"go.yaml.in/yaml/v4"

	"github.com/edelwud/terraci/pkg/config/overwrite"
	"github.com/edelwud/terraci/plugins/policy/internal/domain"
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
	Enabled       bool           `yaml:"enabled" json:"enabled" jsonschema:"description=Enable policy checks,default=false"`
	Sources       []SourceConfig `yaml:"sources,omitempty" json:"sources,omitempty" jsonschema:"description=Policy sources"`
	Namespaces    []string       `yaml:"namespaces,omitempty" json:"namespaces,omitempty" jsonschema:"description=Rego package namespaces to evaluate"`
	FailureAction domain.Action  `yaml:"failure_action,omitempty" json:"failure_action,omitempty" jsonschema:"description=Action for Rego deny decisions,enum=block,enum=warn,enum=ignore,default=block"`
	WarningAction domain.Action  `yaml:"warning_action,omitempty" json:"warning_action,omitempty" jsonschema:"description=Action for Rego warn decisions,enum=block,enum=warn,enum=ignore,default=warn"`
	Overrides     []Override     `yaml:"overrides,omitempty" json:"overrides,omitempty" jsonschema:"description=Per-module policy configuration overrides"`
	CacheDir      string         `yaml:"cache_dir,omitempty" json:"cache_dir,omitempty" jsonschema:"description=Directory to cache materialized policies,default=.terraci/policies"`
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
	Match         string        `yaml:"match" json:"match" jsonschema:"description=Glob pattern to match module paths,required"`
	Enabled       *bool         `yaml:"enabled,omitempty" json:"enabled,omitempty" jsonschema:"description=Override enabled state for matching modules"`
	Namespaces    []string      `yaml:"namespaces,omitempty" json:"namespaces,omitempty" jsonschema:"description=Override namespaces for matching modules"`
	FailureAction domain.Action `yaml:"failure_action,omitempty" json:"failure_action,omitempty" jsonschema:"description=Override action for Rego deny decisions"`
	WarningAction domain.Action `yaml:"warning_action,omitempty" json:"warning_action,omitempty" jsonschema:"description=Override action for Rego warn decisions"`
}

func (c *Config) UnmarshalYAML(unmarshal func(any) error) error {
	if err := rejectKeys(unmarshal, map[string]string{
		"on_failure": "failure_action",
		"on_warning": "warning_action",
		"overwrites": "overrides",
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
	if err := domain.ValidateAction("failure_action", c.FailureAction); err != nil {
		return err
	}
	if err := domain.ValidateAction("warning_action", c.WarningAction); err != nil {
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
	if err := domain.ValidateAction("failure_action", o.FailureAction); err != nil {
		return err
	}
	return domain.ValidateAction("warning_action", o.WarningAction)
}

func (c *Config) EffectiveConfig(modulePath string) (*Config, error) {
	if c == nil {
		return nil, nil
	}

	effective := c.normalized()
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
	if len(c.Namespaces) > 0 {
		return append([]string(nil), c.Namespaces...)
	}
	return []string{DefaultNamespace}
}

func (c *Config) ActionPolicy() domain.ActionPolicy {
	normalized := c.normalized()
	return domain.ActionPolicy{
		FailureAction: normalized.FailureAction,
		WarningAction: normalized.WarningAction,
	}
}

func (c *Config) CanBlock() bool {
	return c.ActionPolicy().CanBlock()
}

func (c *Config) normalized() Config {
	if c == nil {
		return Config{
			FailureAction: domain.ActionBlock,
			WarningAction: domain.ActionWarn,
		}
	}
	out := *c
	out.Sources = append([]SourceConfig(nil), c.Sources...)
	out.Namespaces = append([]string(nil), c.Namespaces...)
	out.Overrides = append([]Override(nil), c.Overrides...)
	if out.FailureAction == "" {
		out.FailureAction = domain.ActionBlock
	}
	if out.WarningAction == "" {
		out.WarningAction = domain.ActionWarn
	}
	return out
}

func applyOverride(effective *Config, override *Override) {
	if override.Enabled != nil {
		effective.Enabled = *override.Enabled
	}
	if len(override.Namespaces) > 0 {
		effective.Namespaces = append([]string(nil), override.Namespaces...)
	}
	if override.FailureAction != "" {
		effective.FailureAction = override.FailureAction
	}
	if override.WarningAction != "" {
		effective.WarningAction = override.WarningAction
	}
}
