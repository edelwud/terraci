package config

import "maps"

const (
	DefaultServiceDir        = ".terraci"
	ExecutionBinaryTerraform = "terraform"
	ExecutionBinaryTofu      = "tofu"
	DefaultParallelism       = 4
)

// Config is the immutable TerraCi configuration read model.
//
//nolint:recvcheck // UnmarshalYAML requires a pointer receiver; read methods intentionally use value receivers.
type Config struct {
	serviceDir     string
	execution      ExecutionConfig
	structure      StructureConfig
	exclude        []string
	include        []string
	libraryModules *LibraryModulesConfig
	extensions     extensionNodeMap
}

// ExecutionConfig defines shared Terraform/OpenTofu execution settings.
type ExecutionConfig struct {
	binary      string
	initEnabled bool
	parallelism int
	env         map[string]string
}

// LibraryModulesConfig defines configuration for library/shared modules
type LibraryModulesConfig struct {
	paths []string
}

// StructureConfig defines the directory structure
type StructureConfig struct {
	pattern  string
	segments PatternSegments
}

// ExecutionConfigOptions describes shared Terraform/OpenTofu execution settings.
type ExecutionConfigOptions struct {
	Binary      string
	InitEnabled *bool
	Parallelism int
	Env         map[string]string
}

// NewExecutionConfig creates immutable execution settings.
func NewExecutionConfig(opts ExecutionConfigOptions) (ExecutionConfig, error) {
	binary := opts.Binary
	if binary == "" {
		binary = ExecutionBinaryTerraform
	}
	switch binary {
	case ExecutionBinaryTerraform, ExecutionBinaryTofu:
	default:
		return ExecutionConfig{}, unsupportedExecutionBinaryError(binary)
	}

	initEnabled := true
	if opts.InitEnabled != nil {
		initEnabled = *opts.InitEnabled
	}

	parallelism := opts.Parallelism
	if parallelism == 0 {
		parallelism = DefaultParallelism
	}
	if parallelism < 1 {
		return ExecutionConfig{}, invalidParallelismError()
	}

	return ExecutionConfig{
		binary:      binary,
		initEnabled: initEnabled,
		parallelism: parallelism,
		env:         maps.Clone(opts.Env),
	}, nil
}

// Binary returns the Terraform-compatible binary name.
func (c ExecutionConfig) Binary() string {
	return c.binary
}

// InitEnabled reports whether terraform init should run before operations.
func (c ExecutionConfig) InitEnabled() bool { return c.initEnabled }

// Parallelism returns local execution parallelism.
func (c ExecutionConfig) Parallelism() int {
	return c.parallelism
}

// Env returns defensive Terraform job environment variables.
func (c ExecutionConfig) Env() map[string]string { return maps.Clone(c.env) }

// StructureConfigOptions describes module directory structure settings.
type StructureConfigOptions struct {
	Pattern string
}

// NewStructureConfig creates immutable structure settings.
func NewStructureConfig(opts StructureConfigOptions) (StructureConfig, error) {
	pattern := opts.Pattern
	if pattern == "" {
		pattern = DefaultPattern
	}
	segments, err := ParsePattern(pattern)
	if err != nil {
		return StructureConfig{}, err
	}
	return StructureConfig{
		pattern:  pattern,
		segments: segments,
	}, nil
}

// Pattern returns the configured module directory pattern.
func (c StructureConfig) Pattern() string {
	return c.pattern
}

// Segments returns defensive parsed pattern segments.
func (c StructureConfig) Segments() PatternSegments {
	return append(PatternSegments(nil), c.segments...)
}

// LibraryModulesConfigOptions describes shared module directory settings.
type LibraryModulesConfigOptions struct {
	Paths []string
}

// NewLibraryModulesConfig creates immutable library module settings.
func NewLibraryModulesConfig(opts LibraryModulesConfigOptions) (LibraryModulesConfig, error) {
	return LibraryModulesConfig{paths: append([]string(nil), opts.Paths...)}, nil
}

// Paths returns defensive library module paths.
func (c LibraryModulesConfig) Paths() []string {
	return append([]string(nil), c.paths...)
}

// ServiceDir returns the project-level service directory for cache and artifacts.
func (c Config) ServiceDir() string {
	return c.serviceDir
}

// Present reports whether this value contains loaded/defaulted config data.
func (c Config) Present() bool {
	return c.serviceDir != "" || c.execution.binary != "" || c.structure.pattern != ""
}

// Execution returns immutable execution settings.
func (c Config) Execution() ExecutionConfig {
	return c.execution.clone()
}

// Structure returns immutable structure settings.
func (c Config) Structure() StructureConfig {
	return c.structure.clone()
}

// Exclude returns defensive exclude patterns.
func (c Config) Exclude() []string {
	return append([]string(nil), c.exclude...)
}

// Include returns defensive include patterns.
func (c Config) Include() []string {
	return append([]string(nil), c.include...)
}

// LibraryModules returns defensive library module settings, if configured.
func (c Config) LibraryModules() *LibraryModulesConfig {
	return cloneLibraryModulesConfig(c.libraryModules)
}
