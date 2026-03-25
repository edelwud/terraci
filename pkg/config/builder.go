package config

import "go.yaml.in/yaml/v4"

// BuildConfigFromPlugins assembles a Config from a pattern and plugin contributions.
func BuildConfigFromPlugins(pattern string, pluginConfigs map[string]map[string]any) *Config {
	cfg := DefaultConfig()
	if pattern != "" {
		cfg.Structure.Pattern = pattern
		if segments, err := ParsePattern(pattern); err == nil {
			cfg.Structure.Segments = segments
		}
	}
	for key, m := range pluginConfigs {
		setPluginNode(cfg, key, m)
	}
	return cfg
}

// setPluginNode marshals a value and stores it in the Plugins map.
func setPluginNode(cfg *Config, key string, value any) {
	data, err := yaml.Marshal(value)
	if err != nil {
		return
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return
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
}

// SetPluginValue is a public helper for setting a plugin config value.
// Used by CLI commands that need to modify plugin configs (e.g., --plan-only).
func SetPluginValue(cfg *Config, pluginKey, field string, value any) {
	// Decode existing config into a map
	m := make(map[string]any)
	if cfg.Plugins != nil {
		if node, ok := cfg.Plugins[pluginKey]; ok {
			if err := node.Decode(&m); err != nil {
				return
			}
		}
	}
	m[field] = value
	setPluginNode(cfg, pluginKey, m)
}
