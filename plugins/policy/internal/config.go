package policyengine

import (
	"errors"
	"fmt"

	"github.com/edelwud/terraci/pkg/config/overwrite"
)

// Config defines configuration for OPA policy checks
type Config struct {
	// Enabled enables policy checks
	Enabled bool `yaml:"enabled" json:"enabled" jsonschema:"description=Enable policy checks,default=false"`

	// Sources defines where to load policies from
	Sources []SourceConfig `yaml:"sources,omitempty" json:"sources,omitempty" jsonschema:"description=Policy sources (local path, git, or OCI)"`

	// Namespaces specifies which Rego packages to evaluate
	Namespaces []string `yaml:"namespaces,omitempty" json:"namespaces,omitempty" jsonschema:"description=Rego package namespaces to evaluate"`

	// OnFailure defines behavior when policy check fails: block or warn
	OnFailure Action `yaml:"on_failure,omitempty" json:"on_failure,omitempty" jsonschema:"description=Behavior on policy failure,enum=block,enum=warn,default=block"`

	// OnWarning defines behavior for warnings: warn or ignore
	OnWarning Action `yaml:"on_warning,omitempty" json:"on_warning,omitempty" jsonschema:"description=Behavior on policy warnings,enum=warn,enum=ignore,default=warn"`

	// Overwrites defines per-module policy overrides
	Overwrites []Overwrite `yaml:"overwrites,omitempty" json:"overwrites,omitempty" jsonschema:"description=Per-module policy configuration overrides"`

	// CacheDir is the directory to cache pulled policies
	CacheDir string `yaml:"cache_dir,omitempty" json:"cache_dir,omitempty" jsonschema:"description=Directory to cache pulled policies,default=.terraci/policies"`
}

// Action defines the action to take on policy violations
type Action string

const (
	// ActionBlock fails the pipeline on violation
	ActionBlock Action = "block"
	// ActionWarn only warns but doesn't fail
	ActionWarn Action = "warn"
	// ActionIgnore ignores the violation
	ActionIgnore Action = "ignore"
)

// SourceConfig defines a source for policy files
type SourceConfig struct {
	// Path is a local path to policies directory
	Path string `yaml:"path,omitempty" json:"path,omitempty" jsonschema:"description=Local path to policies directory"`

	// Git is a git repository URL
	Git string `yaml:"git,omitempty" json:"git,omitempty" jsonschema:"description=Git repository URL for policies"`

	// Ref is the git reference (branch, tag, commit) - used with Git
	Ref string `yaml:"ref,omitempty" json:"ref,omitempty" jsonschema:"description=Git reference (branch, tag, commit)"`

	// OCI is an OCI bundle URL
	OCI string `yaml:"oci,omitempty" json:"oci,omitempty" jsonschema:"description=OCI bundle URL for policies"`
}

// Type returns the type of policy source
func (s *SourceConfig) Type() string {
	if s.Path != "" {
		return "path"
	}
	if s.Git != "" {
		return "git"
	}
	if s.OCI != "" {
		return "oci"
	}
	return ""
}

// Overwrite defines per-module policy overrides
type Overwrite struct {
	// Match is a glob pattern to match module paths
	Match string `yaml:"match" json:"match" jsonschema:"description=Glob pattern to match module paths,required"`

	// Enabled overrides whether policy checks are enabled for matching modules
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty" jsonschema:"description=Override enabled state for matching modules"`

	// Namespaces overrides namespaces for matching modules
	Namespaces []string `yaml:"namespaces,omitempty" json:"namespaces,omitempty" jsonschema:"description=Override namespaces for matching modules"`

	// OnFailure overrides failure behavior for matching modules
	OnFailure Action `yaml:"on_failure,omitempty" json:"on_failure,omitempty" jsonschema:"description=Override failure behavior for matching modules"`

	// OnWarning overrides warning behavior for matching modules
	OnWarning Action `yaml:"on_warning,omitempty" json:"on_warning,omitempty" jsonschema:"description=Override warning behavior for matching modules"`
}

// GetEffectiveConfig returns the effective policy config for a module path
// by applying overwrites that match the path
func (p *Config) GetEffectiveConfig(modulePath string) (*Config, error) {
	if p == nil {
		return nil, nil
	}

	// Start with base config
	effective := Config{
		Enabled:    p.Enabled,
		Namespaces: p.Namespaces,
		OnFailure:  p.OnFailure,
		OnWarning:  p.OnWarning,
		Sources:    p.Sources,
		CacheDir:   p.CacheDir,
	}

	if err := overwrite.ApplyMatching(
		&effective,
		modulePath,
		p.Overwrites,
		overwrite.ByPathGlob(func(ow *Overwrite) string { return ow.Match }),
		applyPolicyOverwrite,
	); err != nil {
		return nil, err
	}

	return &effective, nil
}

func applyPolicyOverwrite(effective *Config, ow *Overwrite) {
	if ow.Enabled != nil {
		effective.Enabled = *ow.Enabled
	}
	if len(ow.Namespaces) > 0 {
		effective.Namespaces = ow.Namespaces
	}
	if ow.OnFailure != "" {
		effective.OnFailure = ow.OnFailure
	}
	if ow.OnWarning != "" {
		effective.OnWarning = ow.OnWarning
	}
}

// Validate checks if the policy configuration is valid
func (p *Config) Validate() error {
	if len(p.Sources) == 0 {
		return errors.New("at least one source is required when policy is enabled")
	}

	for i, src := range p.Sources {
		if src.Type() == "" {
			return fmt.Errorf("sources[%d]: must specify path, git, or oci", i)
		}
	}

	// Validate overwrites
	for i, ow := range p.Overwrites {
		if ow.Match == "" {
			return fmt.Errorf("overwrites[%d].match is required", i)
		}
		if err := overwrite.ValidatePathGlob(ow.Match); err != nil {
			return fmt.Errorf("overwrites[%d].match: %w", i, err)
		}
		if ow.OnFailure != "" && ow.OnFailure != ActionBlock && ow.OnFailure != ActionWarn {
			return fmt.Errorf("overwrites[%d].on_failure must be 'block' or 'warn'", i)
		}
		if ow.OnWarning != "" && ow.OnWarning != ActionWarn && ow.OnWarning != ActionIgnore {
			return fmt.Errorf("overwrites[%d].on_warning must be 'warn' or 'ignore'", i)
		}
	}

	return nil
}
