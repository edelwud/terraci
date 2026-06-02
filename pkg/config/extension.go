package config

import "go.yaml.in/yaml/v4"

// ExtensionDocument is a read-only extension config section captured from
// Config.Extensions.
type ExtensionDocument struct {
	key  ExtensionKey
	node yaml.Node
}

// Extension returns the named extension config document.
func (c *Config) Extension(key ExtensionKey) (ExtensionDocument, bool) {
	if c == nil || c.Extensions == nil || key.String() == "" {
		return ExtensionDocument{}, false
	}
	node, ok := c.Extensions[key.String()]
	if !ok {
		return ExtensionDocument{}, false
	}
	return ExtensionDocument{key: key, node: cloneYAMLNode(node)}, true
}

// Key returns the validated extension key.
func (d ExtensionDocument) Key() ExtensionKey {
	return d.key
}

// Decode decodes the extension config into target.
func (d ExtensionDocument) Decode(target any) error {
	return d.node.Decode(target)
}

// Node returns a defensive copy of the extension YAML node.
func (d ExtensionDocument) Node() yaml.Node {
	return cloneYAMLNode(d.node)
}

// Clone returns a defensive copy of d.
func (d ExtensionDocument) Clone() ExtensionDocument {
	d.node = cloneYAMLNode(d.node)
	return d
}
