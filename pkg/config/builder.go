package config

import (
	"fmt"

	"go.yaml.in/yaml/v4"
)

// BuildConfig assembles a Config from a pattern, execution settings, and
// per-extension configuration sections.
func BuildConfig(pattern string, execution map[string]any, extensions map[string]map[string]any) (*Config, error) {
	cfg := DefaultConfig()
	if pattern != "" {
		cfg.Structure.Pattern = pattern
		if segments, err := ParsePattern(pattern); err == nil {
			cfg.Structure.Segments = segments
		}
	}
	if len(execution) != 0 {
		data, err := yaml.Marshal(execution)
		if err != nil {
			return nil, fmt.Errorf("marshal execution config: %w", err)
		}
		if err := yaml.Unmarshal(data, &cfg.Execution); err != nil {
			return nil, fmt.Errorf("decode execution config: %w", err)
		}
	}
	for key, m := range extensions {
		if err := setExtensionNode(cfg, key, m); err != nil {
			return nil, fmt.Errorf("set extension %q config: %w", key, err)
		}
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}
	return cfg, nil
}

// setExtensionNode marshals a value and stores it in the Extensions map.
func setExtensionNode(cfg *Config, key string, value any) error {
	data, err := yaml.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal extension %q: %w", key, err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("unmarshal extension %q node: %w", key, err)
	}
	if cfg.Extensions == nil {
		cfg.Extensions = make(map[string]yaml.Node)
	}
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		cfg.Extensions[key] = *doc.Content[0]
	} else {
		cfg.Extensions[key] = doc
	}
	return nil
}
