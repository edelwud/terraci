package config

// Default returns TerraCi config with canonical defaults.
func Default() Config {
	return Config{
		serviceDir: DefaultServiceDir,
		execution: ExecutionConfig{
			binary:      ExecutionBinaryTerraform,
			initEnabled: true,
			parallelism: DefaultParallelism,
		},
		structure: StructureConfig{
			pattern:  DefaultPattern,
			segments: DefaultSegments(),
		},
		extensions: make(extensionNodeMap),
	}
}
