package config

import (
	"maps"

	"go.yaml.in/yaml/v4"
)

// Clone returns a deep copy of c. Nil receivers return nil.
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}

	clone := *c
	clone.Execution = c.Execution.clone()
	clone.Structure = c.Structure.clone()
	clone.Exclude = append([]string(nil), c.Exclude...)
	clone.Include = append([]string(nil), c.Include...)
	clone.LibraryModules = c.LibraryModules.clone()
	clone.Extensions = cloneYAMLNodeMap(c.Extensions)
	return &clone
}

func (c ExecutionConfig) clone() ExecutionConfig {
	c.Env = maps.Clone(c.Env)
	return c
}

func (c *LibraryModulesConfig) clone() *LibraryModulesConfig {
	if c == nil {
		return nil
	}
	clone := *c
	clone.Paths = append([]string(nil), c.Paths...)
	return &clone
}

func (c StructureConfig) clone() StructureConfig {
	c.Segments = append(PatternSegments(nil), c.Segments...)
	return c
}

func cloneYAMLNodeMap(nodes map[string]yaml.Node) map[string]yaml.Node {
	if nodes == nil {
		return nil
	}
	clone := make(map[string]yaml.Node, len(nodes))
	for key := range nodes {
		clone[key] = cloneYAMLNode(nodes[key])
	}
	return clone
}

func cloneYAMLNode(node yaml.Node) yaml.Node {
	clone := node
	if node.Content != nil {
		clone.Content = make([]*yaml.Node, len(node.Content))
		for i, child := range node.Content {
			if child == nil {
				continue
			}
			childClone := cloneYAMLNode(*child)
			clone.Content[i] = &childClone
		}
	}
	if node.Alias != nil {
		aliasClone := cloneYAMLNode(*node.Alias)
		clone.Alias = &aliasClone
	}
	return clone
}
