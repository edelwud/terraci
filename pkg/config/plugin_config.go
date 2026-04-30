package config

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
