package config

import "go.yaml.in/yaml/v4"

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		ServiceDir: ".terraci",
		Execution: ExecutionConfig{
			Binary:      ExecutionBinaryTerraform,
			InitEnabled: true,
			PlanEnabled: true,
			PlanMode:    "standard",
			Parallelism: 4,
		},
		Structure: StructureConfig{
			Pattern:  DefaultPattern,
			Segments: DefaultSegments(),
		},
		Extensions: make(map[string]yaml.Node),
	}
}
