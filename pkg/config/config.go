// Package config provides configuration management for terraci
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v4"

	terrierrors "github.com/edelwud/terraci/pkg/errors"
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

	// Plugins holds plugin-specific configuration.
	// Each key is a plugin's ConfigKey(), value is decoded by the plugin.
	Plugins map[string]yaml.Node `yaml:"plugins,omitempty" json:"-" jsonschema:"-"`
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
		return fmt.Errorf("structure.pattern is required")
	}

	if _, err := ParsePattern(c.Structure.Pattern); err != nil {
		return fmt.Errorf("structure.pattern: %w", err)
	}

	return nil
}
