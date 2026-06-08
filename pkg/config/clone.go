package config

import (
	"maps"

	"go.yaml.in/yaml/v4"
)

func (c ExecutionConfig) clone() ExecutionConfig {
	c.env = maps.Clone(c.env)
	return c
}

func cloneLibraryModulesConfig(c *LibraryModulesConfig) *LibraryModulesConfig {
	if c == nil {
		return nil
	}
	clone := *c
	clone.paths = append([]string(nil), c.paths...)
	return &clone
}

func (c StructureConfig) clone() StructureConfig {
	c.segments = append(PatternSegments(nil), c.segments...)
	return c
}

func cloneYAMLNodeMap(nodes extensionNodeMap) extensionNodeMap {
	if nodes == nil {
		return nil
	}
	clone := make(extensionNodeMap, len(nodes))
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
