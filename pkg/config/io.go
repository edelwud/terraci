package config

import (
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v4"

	terrierrors "github.com/edelwud/terraci/pkg/errors"
)

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

	header := fmt.Sprintf("# yaml-language-server: $schema=%s\n", SchemaURL)
	content := append([]byte(header), data...)

	if err := os.WriteFile(path, content, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
