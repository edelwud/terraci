package config

type configYAML struct {
	ServiceDir     string              `yaml:"service_dir,omitempty"`
	Execution      executionYAML       `yaml:"execution,omitempty"`
	Structure      structureYAML       `yaml:"structure"`
	Exclude        []string            `yaml:"exclude,omitempty"`
	Include        []string            `yaml:"include,omitempty"`
	LibraryModules *libraryModulesYAML `yaml:"library_modules,omitempty"`
	Extensions     extensionNodeMap    `yaml:"extensions,omitempty"`
}

type executionYAML struct {
	Binary      string            `yaml:"binary,omitempty"`
	InitEnabled bool              `yaml:"init_enabled,omitempty"`
	Parallelism int               `yaml:"parallelism,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
}

type structureYAML struct {
	Pattern string `yaml:"pattern"`
}

type libraryModulesYAML struct {
	Paths []string `yaml:"paths"`
}

// MarshalYAML preserves the public .terraci.yaml shape while keeping runtime
// config fields private to pkg/config.
func (c Config) MarshalYAML() (any, error) {
	wire := c.toYAML()
	if len(wire.Extensions) == 0 {
		wire.Extensions = nil
	}
	return wire, nil
}

// UnmarshalYAML routes public YAML shape into the immutable runtime value.
func (c *Config) UnmarshalYAML(unmarshal func(any) error) error {
	wire := defaultYAML()
	if err := unmarshal(&wire); err != nil {
		return err
	}
	next, err := configFromYAML(wire)
	if err != nil {
		return err
	}
	*c = next
	return nil
}

func (c Config) toYAML() configYAML {
	return configYAML{
		ServiceDir: c.ServiceDir(),
		Execution: executionYAML{
			Binary:      c.execution.Binary(),
			InitEnabled: c.execution.InitEnabled(),
			Parallelism: c.execution.Parallelism(),
			Env:         c.execution.Env(),
		},
		Structure: structureYAML{
			Pattern: c.structure.Pattern(),
		},
		Exclude: append([]string(nil), c.exclude...),
		Include: append([]string(nil), c.include...),
		LibraryModules: func() *libraryModulesYAML {
			if c.libraryModules == nil {
				return nil
			}
			return &libraryModulesYAML{Paths: c.libraryModules.Paths()}
		}(),
		Extensions: cloneYAMLNodeMap(c.extensions),
	}
}

func defaultYAML() configYAML {
	return Default().toYAML()
}

func configFromYAML(wire configYAML) (Config, error) {
	if wire.Execution.Parallelism < 1 {
		return Config{}, invalidParallelismError()
	}
	execution, err := NewExecutionConfig(ExecutionConfigOptions{
		Binary:      wire.Execution.Binary,
		InitEnabled: &wire.Execution.InitEnabled,
		Parallelism: wire.Execution.Parallelism,
		Env:         wire.Execution.Env,
	})
	if err != nil {
		return Config{}, err
	}

	structure, err := NewStructureConfig(StructureConfigOptions{Pattern: wire.Structure.Pattern})
	if err != nil {
		return Config{}, err
	}

	var libraryModules *LibraryModulesConfig
	if wire.LibraryModules != nil {
		cfg, err := NewLibraryModulesConfig(LibraryModulesConfigOptions{Paths: wire.LibraryModules.Paths})
		if err != nil {
			return Config{}, err
		}
		libraryModules = &cfg
	}

	cfg := Config{
		serviceDir:     wire.ServiceDir,
		execution:      execution,
		structure:      structure,
		exclude:        append([]string(nil), wire.Exclude...),
		include:        append([]string(nil), wire.Include...),
		libraryModules: libraryModules,
		extensions:     cloneYAMLNodeMap(wire.Extensions),
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
