// Package config provides configuration management for terraci
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v4"
)

// Config represents the terraci configuration
type Config struct {
	// Structure defines the directory structure pattern
	Structure StructureConfig `yaml:"structure" json:"structure" jsonschema:"description=Directory structure configuration"`

	// Exclude patterns for modules to ignore
	Exclude []string `yaml:"exclude,omitempty" json:"exclude,omitempty" jsonschema:"description=Glob patterns for modules to exclude"`

	// Include patterns (if set, only matching modules are included)
	Include []string `yaml:"include,omitempty" json:"include,omitempty" jsonschema:"description=Glob patterns for modules to include (if empty, all modules are included after excludes)"`

	// LibraryModules configuration for shared/reusable modules
	LibraryModules *LibraryModulesConfig `yaml:"library_modules,omitempty" json:"library_modules,omitempty" jsonschema:"description=Configuration for library/shared modules (non-executable modules used by other modules)"`

	// GitLab CI configuration
	GitLab GitLabConfig `yaml:"gitlab" json:"gitlab" jsonschema:"description=GitLab CI configuration"`

	// Backend configuration for state file path resolution
	Backend BackendConfig `yaml:"backend" json:"backend" jsonschema:"description=Backend configuration for state file path resolution"`
}

// LibraryModulesConfig defines configuration for library/shared modules
type LibraryModulesConfig struct {
	// Paths is a list of directories containing library modules (relative to root)
	// e.g., ["_modules", "shared/modules"]
	Paths []string `yaml:"paths" json:"paths" jsonschema:"description=List of directories containing library modules (relative to root)"`
}

// StructureConfig defines the directory structure
type StructureConfig struct {
	// Pattern like "{service}/{environment}/{region}/{module}"
	Pattern string `yaml:"pattern" json:"pattern" jsonschema:"description=Pattern describing module directory layout. Supported placeholders: {service}\\, {environment}\\, {region}\\, {module},default={service}/{environment}/{region}/{module}"`
	// MinDepth minimum directory depth (default: 4 for service/env/region/module)
	MinDepth int `yaml:"min_depth,omitempty" json:"min_depth,omitempty" jsonschema:"description=Minimum directory depth for modules (auto-calculated from pattern if not set),minimum=1,default=4"`
	// MaxDepth maximum directory depth (default: 5 for service/env/region/module/submodule)
	MaxDepth int `yaml:"max_depth,omitempty" json:"max_depth,omitempty" jsonschema:"description=Maximum directory depth for modules (allows submodules if > min_depth),minimum=1,default=5"`
	// AllowSubmodules enables nested submodule support
	AllowSubmodules bool `yaml:"allow_submodules" json:"allow_submodules,omitempty" jsonschema:"description=Enable nested submodule support,default=true"`
}

// GitLabConfig contains GitLab CI specific settings
type GitLabConfig struct {
	// TerraformBinary is the terraform binary to use (e.g., "terraform", "tofu")
	TerraformBinary string `yaml:"terraform_binary" json:"terraform_binary" jsonschema:"description=Terraform/OpenTofu binary to use,enum=terraform,enum=tofu,default=terraform"`
	// Image is the Docker image for terraform jobs (in default section)
	// Supports both string format ("hashicorp/terraform:1.6") and object format with entrypoint
	Image Image `yaml:"image" json:"image" jsonschema:"description=Docker image for terraform jobs,default=hashicorp/terraform:1.6"`
	// StagesPrefix is the prefix for stage names (e.g., "deploy" -> "deploy-0", "deploy-1")
	StagesPrefix string `yaml:"stages_prefix" json:"stages_prefix" jsonschema:"description=Prefix for stage names (produces: {prefix}-plan-0\\, {prefix}-apply-0\\, etc.),default=deploy"`
	// Parallelism limits concurrent jobs per stage
	Parallelism int `yaml:"parallelism" json:"parallelism" jsonschema:"description=Maximum parallel jobs per stage,minimum=1,default=5"`
	// Variables to set in the pipeline (global variables section)
	Variables map[string]string `yaml:"variables,omitempty" json:"variables,omitempty" jsonschema:"description=Global pipeline variables"`
	// PlanEnabled enables terraform plan stage
	PlanEnabled bool `yaml:"plan_enabled" json:"plan_enabled" jsonschema:"description=Enable terraform plan stage,default=true"`
	// PlanOnly generates only plan jobs without apply jobs
	PlanOnly bool `yaml:"plan_only" json:"plan_only" jsonschema:"description=Generate only plan jobs (no apply jobs),default=false"`
	// AutoApprove skips manual approval for apply
	AutoApprove bool `yaml:"auto_approve" json:"auto_approve" jsonschema:"description=Auto-approve applies (skip manual confirmation),default=false"`
	// CacheEnabled enables caching of .terraform directory
	CacheEnabled bool `yaml:"cache_enabled" json:"cache_enabled" jsonschema:"description=Enable caching of .terraform directory,default=true"`
	// InitEnabled automatically runs terraform init after cd to module directory
	InitEnabled bool `yaml:"init_enabled" json:"init_enabled" jsonschema:"description=Automatically run terraform init after cd to module directory,default=true"`
	// Rules defines workflow-level rules for conditional pipeline execution
	Rules []Rule `yaml:"rules,omitempty" json:"rules,omitempty" jsonschema:"description=Workflow rules for conditional pipeline execution"`
	// JobDefaults defines default settings for all jobs (applied before overwrites)
	JobDefaults *JobDefaults `yaml:"job_defaults,omitempty" json:"job_defaults,omitempty" jsonschema:"description=Default settings applied to all jobs"`
	// Overwrites defines job-level overrides for plan and apply jobs
	Overwrites []JobOverwrite `yaml:"overwrites,omitempty" json:"overwrites,omitempty" jsonschema:"description=Job-level overrides for plan or apply jobs"`
	// MR contains merge request integration settings
	MR *MRConfig `yaml:"mr,omitempty" json:"mr,omitempty" jsonschema:"description=Merge request integration settings"`
}

// MRConfig contains settings for MR integration
type MRConfig struct {
	// Comment enables MR comment with plan summary (auto-detected in MR pipelines)
	Comment *MRCommentConfig `yaml:"comment,omitempty" json:"comment,omitempty" jsonschema:"description=MR comment configuration"`
	// Labels to add to MR, supports placeholders: {service}, {environment}, {region}, {module}
	Labels []string `yaml:"labels,omitempty" json:"labels,omitempty" jsonschema:"description=Labels to add to MR (supports placeholders: {service}\\, {environment}\\, {region}\\, {module})"`
	// SummaryJob configures the summary job that posts MR comments
	SummaryJob *SummaryJobConfig `yaml:"summary_job,omitempty" json:"summary_job,omitempty" jsonschema:"description=Summary job configuration"`
}

// SummaryJobConfig contains settings for the summary job
type SummaryJobConfig struct {
	// Image for the summary job (must contain terraci binary)
	Image *Image `yaml:"image,omitempty" json:"image,omitempty" jsonschema:"description=Docker image for summary job (must contain terraci)"`
	// Tags for the summary job runner
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty" jsonschema:"description=Runner tags for summary job"`
}

// MRCommentConfig contains settings for MR comments
type MRCommentConfig struct {
	// Enabled enables MR comments (default: true when in MR pipeline)
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty" jsonschema:"description=Enable MR comments,default=true"`
	// OnPlanOnly only comment when there are changes (default: false)
	OnChangesOnly bool `yaml:"on_changes_only,omitempty" json:"on_changes_only,omitempty" jsonschema:"description=Only comment when there are changes"`
	// IncludeDetails includes full plan output in collapsible sections
	IncludeDetails bool `yaml:"include_details,omitempty" json:"include_details,omitempty" jsonschema:"description=Include full plan output in expandable sections,default=true"`
}

// JobConfig is an interface for job configuration (defaults and overwrites)
type JobConfig interface {
	GetImage() *Image
	GetIDTokens() map[string]IDToken
	GetSecrets() map[string]Secret
	GetBeforeScript() []string
	GetAfterScript() []string
	GetArtifacts() *ArtifactsConfig
	GetTags() []string
	GetRules() []Rule
	GetVariables() map[string]string
}

// JobDefaults defines default settings for all generated jobs
type JobDefaults struct {
	// Image overrides the Docker image for all jobs
	Image *Image `yaml:"image,omitempty" json:"image,omitempty" jsonschema:"description=Docker image override for all jobs"`
	// IDTokens sets OIDC tokens for all jobs
	IDTokens map[string]IDToken `yaml:"id_tokens,omitempty" json:"id_tokens,omitempty" jsonschema:"description=OIDC tokens for cloud provider authentication"`
	// Secrets sets secrets for all jobs
	Secrets map[string]Secret `yaml:"secrets,omitempty" json:"secrets,omitempty" jsonschema:"description=Secrets from external secret managers"`
	// BeforeScript sets before_script for all jobs
	BeforeScript []string `yaml:"before_script,omitempty" json:"before_script,omitempty" jsonschema:"description=Commands to run before each job"`
	// AfterScript sets after_script for all jobs
	AfterScript []string `yaml:"after_script,omitempty" json:"after_script,omitempty" jsonschema:"description=Commands to run after each job"`
	// Artifacts sets artifacts configuration for all jobs
	Artifacts *ArtifactsConfig `yaml:"artifacts,omitempty" json:"artifacts,omitempty" jsonschema:"description=GitLab CI artifacts configuration"`
	// Tags sets runner tags for all jobs
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty" jsonschema:"description=GitLab runner tags"`
	// Rules sets job-level rules for all jobs
	Rules []Rule `yaml:"rules,omitempty" json:"rules,omitempty" jsonschema:"description=Job-level rules"`
	// Variables sets additional variables for all jobs
	Variables map[string]string `yaml:"variables,omitempty" json:"variables,omitempty" jsonschema:"description=Additional variables"`
}

// JobDefaults implements JobConfig
func (j *JobDefaults) GetImage() *Image                { return j.Image }
func (j *JobDefaults) GetIDTokens() map[string]IDToken { return j.IDTokens }
func (j *JobDefaults) GetSecrets() map[string]Secret   { return j.Secrets }
func (j *JobDefaults) GetBeforeScript() []string       { return j.BeforeScript }
func (j *JobDefaults) GetAfterScript() []string        { return j.AfterScript }
func (j *JobDefaults) GetArtifacts() *ArtifactsConfig  { return j.Artifacts }
func (j *JobDefaults) GetTags() []string               { return j.Tags }
func (j *JobDefaults) GetRules() []Rule                { return j.Rules }
func (j *JobDefaults) GetVariables() map[string]string { return j.Variables }

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
	Type JobOverwriteType `yaml:"type" json:"type" jsonschema:"description=Type of jobs to override,enum=plan,enum=apply,required"`
	// Image overrides the Docker image for matching jobs
	Image *Image `yaml:"image,omitempty" json:"image,omitempty" jsonschema:"description=Docker image override for matching jobs"`
	// IDTokens overrides OIDC tokens for matching jobs
	IDTokens map[string]IDToken `yaml:"id_tokens,omitempty" json:"id_tokens,omitempty" jsonschema:"description=OIDC tokens for matching jobs"`
	// Secrets overrides secrets for matching jobs
	Secrets map[string]Secret `yaml:"secrets,omitempty" json:"secrets,omitempty" jsonschema:"description=Secrets for matching jobs"`
	// BeforeScript overrides before_script for matching jobs
	BeforeScript []string `yaml:"before_script,omitempty" json:"before_script,omitempty" jsonschema:"description=Commands to run before matching jobs"`
	// AfterScript overrides after_script for matching jobs
	AfterScript []string `yaml:"after_script,omitempty" json:"after_script,omitempty" jsonschema:"description=Commands to run after matching jobs"`
	// Artifacts overrides artifacts configuration for matching jobs
	Artifacts *ArtifactsConfig `yaml:"artifacts,omitempty" json:"artifacts,omitempty" jsonschema:"description=Artifacts configuration for matching jobs"`
	// Tags overrides runner tags for matching jobs
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty" jsonschema:"description=Runner tags for matching jobs"`
	// Rules sets job-level rules for matching jobs
	Rules []Rule `yaml:"rules,omitempty" json:"rules,omitempty" jsonschema:"description=Job-level rules for matching jobs"`
	// Variables overrides variables for matching jobs
	Variables map[string]string `yaml:"variables,omitempty" json:"variables,omitempty" jsonschema:"description=Variables for matching jobs"`
}

// JobOverwrite implements JobConfig
func (j *JobOverwrite) GetImage() *Image                { return j.Image }
func (j *JobOverwrite) GetIDTokens() map[string]IDToken { return j.IDTokens }
func (j *JobOverwrite) GetSecrets() map[string]Secret   { return j.Secrets }
func (j *JobOverwrite) GetBeforeScript() []string       { return j.BeforeScript }
func (j *JobOverwrite) GetAfterScript() []string        { return j.AfterScript }
func (j *JobOverwrite) GetArtifacts() *ArtifactsConfig  { return j.Artifacts }
func (j *JobOverwrite) GetTags() []string               { return j.Tags }
func (j *JobOverwrite) GetRules() []Rule                { return j.Rules }
func (j *JobOverwrite) GetVariables() map[string]string { return j.Variables }

// ArtifactsConfig defines GitLab CI artifacts configuration
type ArtifactsConfig struct {
	// Paths is a list of file/directory paths to include as artifacts
	Paths []string `yaml:"paths,omitempty" json:"paths,omitempty" jsonschema:"description=File/directory paths to include as artifacts"`
	// ExpireIn specifies how long artifacts should be kept (e.g., "1 day", "1 week")
	ExpireIn string `yaml:"expire_in,omitempty" json:"expire_in,omitempty" jsonschema:"description=How long to keep artifacts (e.g. '1 day'\\, '1 week')"`
	// Reports defines artifact reports (e.g., terraform)
	Reports *ArtifactReports `yaml:"reports,omitempty" json:"reports,omitempty" jsonschema:"description=Artifact reports configuration"`
	// Name is the artifact archive name
	Name string `yaml:"name,omitempty" json:"name,omitempty" jsonschema:"description=Artifact archive name"`
	// Untracked includes all untracked files
	Untracked bool `yaml:"untracked,omitempty" json:"untracked,omitempty" jsonschema:"description=Include all untracked files"`
	// When specifies when to upload artifacts: on_success, on_failure, always
	When string `yaml:"when,omitempty" json:"when,omitempty" jsonschema:"description=When to upload artifacts,enum=on_success,enum=on_failure,enum=always"`
	// ExposeAs makes artifacts available in MR UI
	ExposeAs string `yaml:"expose_as,omitempty" json:"expose_as,omitempty" jsonschema:"description=Makes artifacts available in MR UI"`
}

// ArtifactReports defines artifact reports configuration
type ArtifactReports struct {
	// Terraform report paths
	Terraform []string `yaml:"terraform,omitempty" json:"terraform,omitempty" jsonschema:"description=Terraform report paths"`
	// JUnit report paths
	JUnit []string `yaml:"junit,omitempty" json:"junit,omitempty" jsonschema:"description=JUnit report paths"`
	// Cobertura coverage report paths
	Cobertura []string `yaml:"cobertura,omitempty" json:"cobertura,omitempty" jsonschema:"description=Cobertura coverage report paths"`
}

// IDToken defines an OIDC token configuration for GitLab CI
type IDToken struct {
	// Aud is the audience for the token (e.g., AWS ARN, GCP project)
	Aud string `yaml:"aud" json:"aud" jsonschema:"description=Audience for the token,required"`
}

// Rule defines a GitLab CI rule for conditional execution
type Rule struct {
	// If is a condition expression (e.g., "$CI_COMMIT_BRANCH == 'main'")
	If string `yaml:"if,omitempty" json:"if,omitempty" jsonschema:"description=Condition expression"`
	// When specifies when to run (always, never, on_success, manual, delayed)
	When string `yaml:"when,omitempty" json:"when,omitempty" jsonschema:"description=When to run,enum=always,enum=never,enum=on_success,enum=manual,enum=delayed"`
	// Changes specifies file patterns that trigger the rule
	Changes []string `yaml:"changes,omitempty" json:"changes,omitempty" jsonschema:"description=File patterns that trigger the rule"`
}

// Secret defines a CI/CD secret from an external secret manager
type Secret struct {
	// Vault configures HashiCorp Vault secret (can be string shorthand or object)
	Vault *VaultSecret `yaml:"vault,omitempty" json:"vault,omitempty" jsonschema:"description=HashiCorp Vault secret configuration"`
	// File indicates if secret should be written to a file
	File bool `yaml:"file,omitempty" json:"file,omitempty" jsonschema:"description=Write secret to a file"`
}

// VaultSecret defines a secret from HashiCorp Vault
// Supports both full object syntax and string shorthand (path/to/secret/field@namespace)
type VaultSecret struct {
	// Engine is the secrets engine (e.g., "kv-v2") - for full syntax
	Engine *VaultEngine `yaml:"engine,omitempty" json:"engine,omitempty" jsonschema:"description=Vault secrets engine configuration"`
	// Path is the path to the secret in Vault - for full syntax
	Path string `yaml:"path,omitempty" json:"path,omitempty" jsonschema:"description=Path to the secret in Vault"`
	// Field is the field to extract from the secret - for full syntax
	Field string `yaml:"field,omitempty" json:"field,omitempty" jsonschema:"description=Field to extract from the secret"`
	// Shorthand is the string shorthand format (path/to/secret/field@namespace)
	Shorthand string `yaml:"-" jsonschema:"-"`
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
	Name string `yaml:"name" json:"name" jsonschema:"description=Engine name"`
	// Path is the engine mount path
	Path string `yaml:"path" json:"path" jsonschema:"description=Engine mount path"`
}

// Image defines a Docker image configuration
// Supports both string format and object format with entrypoint
type Image struct {
	// Name is the image name (e.g., "hashicorp/terraform:1.6")
	Name string `yaml:"name,omitempty" json:"name,omitempty" jsonschema:"description=Docker image name"`
	// Entrypoint overrides the default entrypoint
	Entrypoint []string `yaml:"entrypoint,omitempty" json:"entrypoint,omitempty" jsonschema:"description=Override default entrypoint"`
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
	Type string `yaml:"type" jsonschema:"description=Type of backend,enum=s3,enum=gcs,enum=azurerm,enum=local,enum=remote"`
	// Bucket name for S3/GCS
	Bucket string `yaml:"bucket,omitempty" jsonschema:"description=Bucket name for S3/GCS"`
	// Region for S3
	Region string `yaml:"region,omitempty" jsonschema:"description=Region for S3"`
	// KeyPattern is the pattern for state file keys
	// Supports variables: {service}, {environment}, {region}, {module}
	KeyPattern string `yaml:"key_pattern,omitempty" jsonschema:"description=Pattern for state file keys. Supports variables: {service}\\, {environment}\\, {region}\\, {module},default={service}/{environment}/{region}/{module}/terraform.tfstate"`
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

// GetImage returns the configured image
func (g *GitLabConfig) GetImage() Image {
	return g.Image
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

// SchemaURL is the URL to the JSON Schema for terraci configuration
const SchemaURL = "https://raw.githubusercontent.com/edelwud/terraci/main/.terraci.schema.json"

// Save writes configuration to a file with yaml-language-server schema reference
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Add yaml-language-server schema reference header
	header := fmt.Sprintf("# yaml-language-server: $schema=%s\n", SchemaURL)
	content := append([]byte(header), data...)

	if err := os.WriteFile(path, content, 0o600); err != nil {
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

	if c.GitLab.Image.Name == "" {
		return fmt.Errorf("gitlab.image is required")
	}

	// Validate overwrites
	for i := range c.GitLab.Overwrites {
		if c.GitLab.Overwrites[i].Type != OverwriteTypePlan && c.GitLab.Overwrites[i].Type != OverwriteTypeApply {
			return fmt.Errorf("gitlab.overwrites[%d].type must be 'plan' or 'apply'", i)
		}
	}

	return nil
}
