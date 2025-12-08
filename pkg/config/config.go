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

	// GitLab CI configuration
	GitLab GitLabConfig `yaml:"gitlab"`

	// Backend configuration for state file path resolution
	Backend BackendConfig `yaml:"backend"`
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
	// TerraformImage is the Docker image for terraform jobs
	TerraformImage string `yaml:"terraform_image"`
	// StagesPrefix is the prefix for stage names (e.g., "deploy" -> "deploy-0", "deploy-1")
	StagesPrefix string `yaml:"stages_prefix"`
	// Parallelism limits concurrent jobs per stage
	Parallelism int `yaml:"parallelism"`
	// BeforeScript commands to run before each job
	BeforeScript []string `yaml:"before_script,omitempty"`
	// AfterScript commands to run after each job
	AfterScript []string `yaml:"after_script,omitempty"`
	// Variables to set in the pipeline
	Variables map[string]string `yaml:"variables,omitempty"`
	// Tags for runners
	Tags []string `yaml:"tags,omitempty"`
	// PlanEnabled enables terraform plan stage
	PlanEnabled bool `yaml:"plan_enabled"`
	// AutoApprove skips manual approval for apply
	AutoApprove bool `yaml:"auto_approve"`
	// ArtifactPaths for terraform plans
	ArtifactPaths []string `yaml:"artifact_paths,omitempty"`
	// CacheEnabled enables caching of .terraform directory
	CacheEnabled bool `yaml:"cache_enabled"`
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
			TerraformImage:  "hashicorp/terraform:1.6",
			StagesPrefix:    "deploy",
			Parallelism:     5,
			PlanEnabled:     true,
			AutoApprove:     false,
			BeforeScript: []string{
				"${TERRAFORM_BINARY} init",
			},
			ArtifactPaths: []string{
				"*.tfplan",
			},
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

	if c.GitLab.TerraformImage == "" {
		return fmt.Errorf("gitlab.terraform_image is required")
	}

	return nil
}
