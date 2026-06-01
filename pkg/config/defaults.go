package config

import "go.yaml.in/yaml/v4"

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		ServiceDir: DefaultServiceDir,
		Execution: ExecutionConfig{
			Binary:      ExecutionBinaryTerraform,
			InitEnabled: true,
			Parallelism: 4,
		},
		Structure: StructureConfig{
			Pattern:  DefaultPattern,
			Segments: DefaultSegments(),
		},
		Extensions: make(map[string]yaml.Node),
	}
}
