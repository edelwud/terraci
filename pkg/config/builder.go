package config

import (
	"fmt"

	"go.yaml.in/yaml/v4"
)

// BuildConfigFromPlugins assembles a Config from a pattern and plugin contributions.
func BuildConfigFromPlugins(pattern string, pluginConfigs map[string]map[string]any) (*Config, error) {
	cfg := DefaultConfig()
	if pattern != "" {
		cfg.Structure.Pattern = pattern
		if segments, err := ParsePattern(pattern); err == nil {
			cfg.Structure.Segments = segments
		}
	}
	for key, m := range pluginConfigs {
		if err := setPluginNode(cfg, key, m); err != nil {
			return nil, fmt.Errorf("set plugin %q config: %w", key, err)
		}
	}
	return cfg, nil
}

// setPluginNode marshals a value and stores it in the Plugins map.
func setPluginNode(cfg *Config, key string, value any) error {
	data, err := yaml.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal plugin %q: %w", key, err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("unmarshal plugin %q node: %w", key, err)
	}
	if cfg.Plugins == nil {
		cfg.Plugins = make(map[string]yaml.Node)
	}
	// yaml.Unmarshal wraps in a document node — unwrap to get the mapping node
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		cfg.Plugins[key] = *doc.Content[0]
	} else {
		cfg.Plugins[key] = doc
	}
	return nil
}
