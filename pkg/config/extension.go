package config

// Extension decodes the named extension's configuration section into target.
// Returns nil if the section is missing — callers should use their own defaults.
func (c *Config) Extension(key string, target any) error {
	if c.Extensions == nil {
		return nil
	}
	node, ok := c.Extensions[key]
	if !ok {
		return nil
	}
	return node.Decode(target)
}
