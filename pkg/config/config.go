// Package config provides configuration management for terraci
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the terraci configuration
type Config struct {
	// Structure defines the directory structure pattern
	Structure StructureConfig `yaml:"structure"`

	// Exclude patterns for modules to ignore
	Exclude []string `yaml:"exclude,omitempty"`

	// Include patterns (if set, only matching modules are included)
	Include []string `yaml:"include,omitempty"`

	// LibraryModules configuration for shared/reusable modules
	LibraryModules *LibraryModulesConfig `yaml:"library_modules,omitempty"`

	// GitLab CI configuration
	GitLab GitLabConfig `yaml:"gitlab"`

	// Backend configuration for state file path resolution
	Backend BackendConfig `yaml:"backend"`
}

// LibraryModulesConfig defines configuration for library/shared modules
type LibraryModulesConfig struct {
	// Paths is a list of directories containing library modules (relative to root)
	// e.g., ["_modules", "shared/modules"]
	Paths []string `yaml:"paths"`
}

// StructureConfig defines the directory structure
type StructureConfig struct {
	// Pattern like "{service}/{environment}/{region}/{module}"
	Pattern string `yaml:"pattern"`
	// MinDepth minimum directory depth (default: 4 for service/env/region/module)
	MinDepth int `yaml:"min_depth,omitempty"`
	// MaxDepth maximum directory depth (default: 5 for service/env/region/module/submodule)
	MaxDepth int `yaml:"max_depth,omitempty"`
	// AllowSubmodules enables nested submodule support
	AllowSubmodules bool `yaml:"allow_submodules"`
}

// GitLabConfig contains GitLab CI specific settings
type GitLabConfig struct {
	// TerraformBinary is the terraform binary to use (e.g., "terraform", "tofu")
	TerraformBinary string `yaml:"terraform_binary"`
	// Image is the Docker image for terraform jobs (in default section)
	// Supports both string format ("hashicorp/terraform:1.6") and object format with entrypoint
	Image Image `yaml:"image"`
	// TerraformImage is deprecated, use Image instead
	TerraformImage Image `yaml:"terraform_image,omitempty"`
	// StagesPrefix is the prefix for stage names (e.g., "deploy" -> "deploy-0", "deploy-1")
	StagesPrefix string `yaml:"stages_prefix"`
	// Parallelism limits concurrent jobs per stage
	Parallelism int `yaml:"parallelism"`
	// Variables to set in the pipeline (global variables section)
	Variables map[string]string `yaml:"variables,omitempty"`
	// PlanEnabled enables terraform plan stage
	PlanEnabled bool `yaml:"plan_enabled"`
	// AutoApprove skips manual approval for apply
	AutoApprove bool `yaml:"auto_approve"`
	// CacheEnabled enables caching of .terraform directory
	CacheEnabled bool `yaml:"cache_enabled"`
	// InitEnabled automatically runs terraform init after cd to module directory
	InitEnabled bool `yaml:"init_enabled"`
	// Rules defines workflow-level rules for conditional pipeline execution
	Rules []Rule `yaml:"rules,omitempty"`
	// JobDefaults defines default settings for all jobs (applied before overwrites)
	JobDefaults *JobDefaults `yaml:"job_defaults,omitempty"`
	// Overwrites defines job-level overrides for plan and apply jobs
	Overwrites []JobOverwrite `yaml:"overwrites,omitempty"`
}

// JobDefaults defines default settings for all generated jobs
type JobDefaults struct {
	// Image overrides the Docker image for all jobs
	Image *Image `yaml:"image,omitempty"`
	// IDTokens sets OIDC tokens for all jobs
	IDTokens map[string]IDToken `yaml:"id_tokens,omitempty"`
	// Secrets sets secrets for all jobs
	Secrets map[string]Secret `yaml:"secrets,omitempty"`
	// BeforeScript sets before_script for all jobs
	BeforeScript []string `yaml:"before_script,omitempty"`
	// AfterScript sets after_script for all jobs
	AfterScript []string `yaml:"after_script,omitempty"`
	// Artifacts sets artifacts configuration for all jobs
	Artifacts *ArtifactsConfig `yaml:"artifacts,omitempty"`
	// Tags sets runner tags for all jobs
	Tags []string `yaml:"tags,omitempty"`
	// Rules sets job-level rules for all jobs
	Rules []Rule `yaml:"rules,omitempty"`
	// Variables sets additional variables for all jobs
	Variables map[string]string `yaml:"variables,omitempty"`
}

// JobOverwriteType defines the type of jobs to override
type JobOverwriteType string

const (
	// OverwriteTypePlan applies to plan jobs only
	OverwriteTypePlan JobOverwriteType = "plan"
	// OverwriteTypeApply applies to apply jobs only
	OverwriteTypeApply JobOverwriteType = "apply"
)

// JobOverwrite defines job-level overrides for plan or apply jobs
type JobOverwrite struct {
	// Type specifies which jobs to override: "plan" or "apply"
	Type JobOverwriteType `yaml:"type"`
	// Image overrides the Docker image for matching jobs
	Image *Image `yaml:"image,omitempty"`
	// IDTokens overrides OIDC tokens for matching jobs
	IDTokens map[string]IDToken `yaml:"id_tokens,omitempty"`
	// Secrets overrides secrets for matching jobs
	Secrets map[string]Secret `yaml:"secrets,omitempty"`
	// BeforeScript overrides before_script for matching jobs
	BeforeScript []string `yaml:"before_script,omitempty"`
	// AfterScript overrides after_script for matching jobs
	AfterScript []string `yaml:"after_script,omitempty"`
	// Artifacts overrides artifacts configuration for matching jobs
	Artifacts *ArtifactsConfig `yaml:"artifacts,omitempty"`
	// Tags overrides runner tags for matching jobs
	Tags []string `yaml:"tags,omitempty"`
	// Rules sets job-level rules for matching jobs
	Rules []Rule `yaml:"rules,omitempty"`
	// Variables overrides variables for matching jobs
	Variables map[string]string `yaml:"variables,omitempty"`
}

// ArtifactsConfig defines GitLab CI artifacts configuration
type ArtifactsConfig struct {
	// Paths is a list of file/directory paths to include as artifacts
	Paths []string `yaml:"paths,omitempty"`
	// ExpireIn specifies how long artifacts should be kept (e.g., "1 day", "1 week")
	ExpireIn string `yaml:"expire_in,omitempty"`
	// Reports defines artifact reports (e.g., terraform)
	Reports *ArtifactReports `yaml:"reports,omitempty"`
	// Name is the artifact archive name
	Name string `yaml:"name,omitempty"`
	// Untracked includes all untracked files
	Untracked bool `yaml:"untracked,omitempty"`
	// When specifies when to upload artifacts: on_success, on_failure, always
	When string `yaml:"when,omitempty"`
	// ExposeAs makes artifacts available in MR UI
	ExposeAs string `yaml:"expose_as,omitempty"`
}

// ArtifactReports defines artifact reports configuration
type ArtifactReports struct {
	// Terraform report paths
	Terraform []string `yaml:"terraform,omitempty"`
	// JUnit report paths
	JUnit []string `yaml:"junit,omitempty"`
	// Cobertura coverage report paths
	Cobertura []string `yaml:"cobertura,omitempty"`
}

// IDToken defines an OIDC token configuration for GitLab CI
type IDToken struct {
	// Aud is the audience for the token (e.g., AWS ARN, GCP project)
	Aud string `yaml:"aud"`
}

// Rule defines a GitLab CI rule for conditional execution
type Rule struct {
	// If is a condition expression (e.g., "$CI_COMMIT_BRANCH == 'main'")
	If string `yaml:"if,omitempty"`
	// When specifies when to run (always, never, on_success, manual, delayed)
	When string `yaml:"when,omitempty"`
	// Changes specifies file patterns that trigger the rule
	Changes []string `yaml:"changes,omitempty"`
}

// Secret defines a CI/CD secret from an external secret manager
type Secret struct {
	// Vault configures HashiCorp Vault secret (can be string shorthand or object)
	Vault *VaultSecret `yaml:"vault,omitempty"`
	// File indicates if secret should be written to a file
	File bool `yaml:"file,omitempty"`
}

// VaultSecret defines a secret from HashiCorp Vault
// Supports both full object syntax and string shorthand (path/to/secret/field@namespace)
type VaultSecret struct {
	// Engine is the secrets engine (e.g., "kv-v2") - for full syntax
	Engine *VaultEngine `yaml:"engine,omitempty"`
	// Path is the path to the secret in Vault - for full syntax
	Path string `yaml:"path,omitempty"`
	// Field is the field to extract from the secret - for full syntax
	Field string `yaml:"field,omitempty"`
	// Shorthand is the string shorthand format (path/to/secret/field@namespace)
	Shorthand string `yaml:"-"`
}

// UnmarshalYAML implements custom unmarshaling for VaultSecret to support string shorthand
func (v *VaultSecret) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Try string shorthand first
	var shorthand string
	if err := unmarshal(&shorthand); err == nil {
		v.Shorthand = shorthand
		return nil
	}

	// Try full object syntax
	type vaultSecretAlias VaultSecret
	var alias vaultSecretAlias
	if err := unmarshal(&alias); err != nil {
		return err
	}
	*v = VaultSecret(alias)
	return nil
}

// VaultEngine defines Vault secrets engine configuration
type VaultEngine struct {
	// Name is the engine name
	Name string `yaml:"name"`
	// Path is the engine mount path
	Path string `yaml:"path"`
}

// Image defines a Docker image configuration
// Supports both string format and object format with entrypoint
type Image struct {
	// Name is the image name (e.g., "hashicorp/terraform:1.6")
	Name string `yaml:"name,omitempty"`
	// Entrypoint overrides the default entrypoint
	Entrypoint []string `yaml:"entrypoint,omitempty"`
}

// UnmarshalYAML implements custom unmarshaling for Image to support string shorthand
func (img *Image) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Try string shorthand first (just image name)
	var shorthand string
	if err := unmarshal(&shorthand); err == nil {
		img.Name = shorthand
		return nil
	}

	// Try full object syntax
	type imageAlias Image
	var alias imageAlias
	if err := unmarshal(&alias); err != nil {
		return err
	}
	*img = Image(alias)
	return nil
}

// String returns the image name
func (img *Image) String() string {
	return img.Name
}

// HasEntrypoint returns true if entrypoint is configured
func (img *Image) HasEntrypoint() bool {
	return len(img.Entrypoint) > 0
}

// BackendConfig defines the state backend configuration
type BackendConfig struct {
	// Type of backend (s3, gcs, azurerm, etc.)
	Type string `yaml:"type"`
	// Bucket name for S3/GCS
	Bucket string `yaml:"bucket,omitempty"`
	// Region for S3
	Region string `yaml:"region,omitempty"`
	// KeyPattern is the pattern for state file keys
	// Supports variables: {service}, {environment}, {region}, {module}
	KeyPattern string `yaml:"key_pattern,omitempty"`
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Structure: StructureConfig{
			Pattern:         "{service}/{environment}/{region}/{module}",
			MinDepth:        4,
			MaxDepth:        5,
			AllowSubmodules: true,
		},
		GitLab: GitLabConfig{
			TerraformBinary: "terraform",
			Image:           Image{Name: "hashicorp/terraform:1.6"},
			StagesPrefix:    "deploy",
			Parallelism:     5,
			PlanEnabled:     true,
			AutoApprove:     false,
			InitEnabled:     true,
		},
		Backend: BackendConfig{
			Type:       "s3",
			KeyPattern: "{service}/{environment}/{region}/{module}/terraform.tfstate",
		},
	}
}

// Load reads configuration from a file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Calculate depths from pattern if not set
	if config.Structure.MinDepth == 0 {
		config.Structure.MinDepth = countPatternSegments(config.Structure.Pattern)
	}
	if config.Structure.MaxDepth == 0 {
		if config.Structure.AllowSubmodules {
			config.Structure.MaxDepth = config.Structure.MinDepth + 1
		} else {
			config.Structure.MaxDepth = config.Structure.MinDepth
		}
	}

	return config, nil
}

// GetImage returns the effective image (new field or deprecated terraform_image)
func (g *GitLabConfig) GetImage() Image {
	if g.Image.Name != "" {
		return g.Image
	}
	return g.TerraformImage
}

// LoadOrDefault loads config from file or returns default if not found
func LoadOrDefault(dir string) (*Config, error) {
	configPaths := []string{
		filepath.Join(dir, ".terraci.yaml"),
		filepath.Join(dir, ".terraci.yml"),
		filepath.Join(dir, "terraci.yaml"),
		filepath.Join(dir, "terraci.yml"),
	}

	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			return Load(path)
		}
	}

	return DefaultConfig(), nil
}

// Save writes configuration to a file
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// countPatternSegments counts the number of segments in a pattern
func countPatternSegments(pattern string) int {
	count := 1
	for _, c := range pattern {
		if c == '/' {
			count++
		}
	}
	return count
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Structure.Pattern == "" {
		return fmt.Errorf("structure.pattern is required")
	}

	if c.Structure.MinDepth < 1 {
		return fmt.Errorf("structure.min_depth must be at least 1")
	}

	if c.Structure.MaxDepth < c.Structure.MinDepth {
		return fmt.Errorf("structure.max_depth must be >= min_depth")
	}

	// Check image (prefer new field, fall back to deprecated)
	if c.GitLab.Image.Name == "" && c.GitLab.TerraformImage.Name == "" {
		return fmt.Errorf("gitlab.image is required")
	}

	// Validate overwrites
	for i, ow := range c.GitLab.Overwrites {
		if ow.Type != OverwriteTypePlan && ow.Type != OverwriteTypeApply {
			return fmt.Errorf("gitlab.overwrites[%d].type must be 'plan' or 'apply'", i)
		}
	}

	return nil
}
