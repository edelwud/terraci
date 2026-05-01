package config

import (
	"maps"

	"go.yaml.in/yaml/v4"
)

func cloneYAMLNode(node yaml.Node) yaml.Node {
	cloned := node
	if len(node.Content) > 0 {
		cloned.Content = make([]*yaml.Node, len(node.Content))
		for i, child := range node.Content {
			if child == nil {
				continue
			}
			childClone := cloneYAMLNode(*child)
			cloned.Content[i] = &childClone
		}
	}
	return cloned
}

// Clone returns a deep copy of the configuration.
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}

	cloned := *c
	if c.Structure.Segments != nil {
		cloned.Structure.Segments = append(PatternSegments(nil), c.Structure.Segments...)
	}
	if c.Execution.Env != nil {
		cloned.Execution.Env = make(map[string]string, len(c.Execution.Env))
		maps.Copy(cloned.Execution.Env, c.Execution.Env)
	}
	if c.Exclude != nil {
		cloned.Exclude = append([]string(nil), c.Exclude...)
	}
	if c.Include != nil {
		cloned.Include = append([]string(nil), c.Include...)
	}
	if c.LibraryModules != nil {
		libraryModules := *c.LibraryModules
		if c.LibraryModules.Paths != nil {
			libraryModules.Paths = append([]string(nil), c.LibraryModules.Paths...)
		}
		cloned.LibraryModules = &libraryModules
	}
	if c.Extensions != nil {
		cloned.Extensions = make(map[string]yaml.Node, len(c.Extensions))
		for key := range c.Extensions {
			cloned.Extensions[key] = cloneYAMLNode(c.Extensions[key])
		}
	}

	return &cloned
}
