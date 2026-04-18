// Package config provides configuration management for terraci
package config

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v4"

	terrierrors "github.com/edelwud/terraci/pkg/errors"
)

func cloneYAMLNode(node yaml.Node) yaml.Node {
	cloned := node
	if len(node.Content) > 0 {
		cloned.Content = make([]*yaml.Node, len(node.Content))
		for i, child := range node.Content {
			if child == nil {
				continue
			}
			childClone := cloneYAMLNode(*child)
			cloned.Content[i] = &childClone
		}
	}
	return cloned
}

// Config represents the terraci configuration
type Config struct {
	// ServiceDir is the project-level service directory for cache and artifacts.
	ServiceDir string `yaml:"service_dir,omitempty" json:"service_dir,omitempty" jsonschema:"description=Service directory for cache and artifacts,default=.terraci"`

	// Execution defines shared Terraform/OpenTofu execution semantics.
	Execution ExecutionConfig `yaml:"execution,omitempty" json:"execution,omitempty" jsonschema:"description=Shared execution settings for Terraform/OpenTofu"` //nolint:modernize // yaml/v4 does not support omitzero

	// Structure defines the directory structure pattern
	Structure StructureConfig `yaml:"structure" json:"structure" jsonschema:"description=Directory structure configuration"`

	// Exclude patterns for modules to ignore
	Exclude []string `yaml:"exclude,omitempty" json:"exclude,omitempty" jsonschema:"description=Glob patterns for modules to exclude"`

	// Include patterns (if set, only matching modules are included)
	Include []string `yaml:"include,omitempty" json:"include,omitempty" jsonschema:"description=Glob patterns for modules to include (if empty, all modules are included after excludes)"`

	// LibraryModules configuration for shared/reusable modules
	LibraryModules *LibraryModulesConfig `yaml:"library_modules,omitempty" json:"library_modules,omitempty" jsonschema:"description=Configuration for library/shared modules (non-executable modules used by other modules)"`

	// Plugins holds plugin-specific configuration.
	// Each key is a plugin's ConfigKey(), value is decoded by the plugin.
	Plugins map[string]yaml.Node `yaml:"plugins,omitempty" json:"-" jsonschema:"-"`
}

// ExecutionConfig defines shared Terraform/OpenTofu execution settings.
type ExecutionConfig struct {
	Binary      string            `yaml:"binary,omitempty" json:"binary,omitempty" jsonschema:"description=Terraform/OpenTofu binary to use,enum=terraform,enum=tofu,default=terraform"`
	InitEnabled bool              `yaml:"init_enabled,omitempty" json:"init_enabled,omitempty" jsonschema:"description=Automatically run terraform init before terraform operations,default=true"`
	PlanEnabled bool              `yaml:"plan_enabled,omitempty" json:"plan_enabled,omitempty" jsonschema:"description=Enable terraform plan jobs,default=true"`
	PlanMode    string            `yaml:"plan_mode,omitempty" json:"plan_mode,omitempty" jsonschema:"description=Controls plan artifact verbosity. standard writes only plan.tfplan; detailed also writes plan.txt and plan.json for summary, policy, cost, and PR/MR comment flows,enum=standard,enum=detailed,default=standard"`
	Parallelism int               `yaml:"parallelism,omitempty" json:"parallelism,omitempty" jsonschema:"description=Maximum parallel jobs for local execution,minimum=1,default=4"`
	Env         map[string]string `yaml:"env,omitempty" json:"env,omitempty" jsonschema:"description=Execution-wide environment variables"`
}

// PluginConfig decodes plugin-specific configuration into the target struct.
// Returns nil if the plugin has no configuration (defaults should be used).
func (c *Config) PluginConfig(key string, target any) error {
	if c.Plugins == nil {
		return nil
	}
	node, ok := c.Plugins[key]
	if !ok {
		return nil
	}
	return node.Decode(target)
}

// Clone returns a deep copy of the configuration.
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}

	cloned := *c
	if c.Structure.Segments != nil {
		cloned.Structure.Segments = append(PatternSegments(nil), c.Structure.Segments...)
	}
	if c.Execution.Env != nil {
		cloned.Execution.Env = make(map[string]string, len(c.Execution.Env))
		maps.Copy(cloned.Execution.Env, c.Execution.Env)
	}
	if c.Exclude != nil {
		cloned.Exclude = append([]string(nil), c.Exclude...)
	}
	if c.Include != nil {
		cloned.Include = append([]string(nil), c.Include...)
	}
	if c.LibraryModules != nil {
		libraryModules := *c.LibraryModules
		if c.LibraryModules.Paths != nil {
			libraryModules.Paths = append([]string(nil), c.LibraryModules.Paths...)
		}
		cloned.LibraryModules = &libraryModules
	}
	if c.Plugins != nil {
		cloned.Plugins = make(map[string]yaml.Node, len(c.Plugins))
		for key := range c.Plugins {
			cloned.Plugins[key] = cloneYAMLNode(c.Plugins[key])
		}
	}

	return &cloned
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
	// Segments is the parsed pattern segments (derived from Pattern, not serialized)
	Segments PatternSegments `yaml:"-" json:"-"`
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		ServiceDir: ".terraci",
		Execution: ExecutionConfig{
			Binary:      "terraform",
			InitEnabled: true,
			PlanEnabled: true,
			PlanMode:    "standard",
			Parallelism: 4,
		},
		Structure: StructureConfig{
			Pattern:  "{service}/{environment}/{region}/{module}",
			Segments: PatternSegments{"service", "environment", "region", "module"},
		},
		Plugins: make(map[string]yaml.Node),
	}
}

// Load reads configuration from a file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &terrierrors.ConfigError{Path: path, Err: fmt.Errorf("read: %w", err)}
	}

	cfg := DefaultConfig()
	if unmarshalErr := yaml.Unmarshal(data, cfg); unmarshalErr != nil {
		return nil, &terrierrors.ConfigError{Path: path, Err: fmt.Errorf("parse: %w", unmarshalErr)}
	}

	segments, parseErr := ParsePattern(cfg.Structure.Pattern)
	if parseErr == nil {
		cfg.Structure.Segments = segments
	}

	if err := cfg.Validate(); err != nil {
		return nil, &terrierrors.ConfigError{Path: path, Err: err}
	}

	return cfg, nil
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

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Structure.Pattern == "" {
		return errors.New("structure.pattern is required")
	}

	if _, err := ParsePattern(c.Structure.Pattern); err != nil {
		return fmt.Errorf("structure.pattern: %w", err)
	}

	switch c.Execution.Binary {
	case "", "terraform", "tofu": //nolint:goconst // validation literal, not worth extracting
	default:
		return fmt.Errorf("execution.binary: unsupported value %q", c.Execution.Binary)
	}

	switch c.Execution.PlanMode {
	case "", "standard", "detailed":
	default:
		return fmt.Errorf("execution.plan_mode: unsupported value %q", c.Execution.PlanMode)
	}

	if c.Execution.Parallelism < 0 {
		return errors.New("execution.parallelism: must be >= 0")
	}

	return nil
}
