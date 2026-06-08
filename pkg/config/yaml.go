package config

type configYAML struct {
	ServiceDir     string                `yaml:"service_dir,omitempty"`
	Execution      ExecutionConfig       `yaml:"execution,omitempty"`
	Structure      StructureConfig       `yaml:"structure"`
	Exclude        []string              `yaml:"exclude,omitempty"`
	Include        []string              `yaml:"include,omitempty"`
	LibraryModules *LibraryModulesConfig `yaml:"library_modules,omitempty"`
	Extensions     extensionNodeMap      `yaml:"extensions,omitempty"`
}

// MarshalYAML preserves the public .terraci.yaml shape while keeping extension
// storage private to pkg/config.
func (c *Config) MarshalYAML() (any, error) {
	if c == nil {
		return nil, nil
	}
	wire := c.toYAML()
	if len(wire.Extensions) == 0 {
		wire.Extensions = nil
	}
	return wire, nil
}

// UnmarshalYAML preserves existing defaults when callers decode into
// DefaultConfig(), while routing raw extension nodes into private storage.
func (c *Config) UnmarshalYAML(unmarshal func(any) error) error {
	wire := c.toYAML()
	if err := unmarshal(&wire); err != nil {
		return err
	}
	c.ServiceDir = wire.ServiceDir
	c.Execution = wire.Execution
	c.Structure = wire.Structure
	c.Exclude = append([]string(nil), wire.Exclude...)
	c.Include = append([]string(nil), wire.Include...)
	c.LibraryModules = wire.LibraryModules.clone()
	c.extensions = cloneYAMLNodeMap(wire.Extensions)
	return nil
}

func (c *Config) toYAML() configYAML {
	if c == nil {
		return configYAML{}
	}
	return configYAML{
		ServiceDir:     c.ServiceDir,
		Execution:      c.Execution.clone(),
		Structure:      c.Structure.clone(),
		Exclude:        append([]string(nil), c.Exclude...),
		Include:        append([]string(nil), c.Include...),
		LibraryModules: c.LibraryModules.clone(),
		Extensions:     cloneYAMLNodeMap(c.extensions),
	}
}
