package config

import "go.yaml.in/yaml/v4"

const (
	ExecutionBinaryTerraform = "terraform"
	ExecutionBinaryTofu      = "tofu"
)

// Config represents the terraci configuration
type Config struct {
	// ServiceDir is the project-level service directory for cache and artifacts.
	ServiceDir string `yaml:"service_dir,omitempty" json:"service_dir,omitempty" jsonschema:"description=Service directory for cache and artifacts,default=.terraci"`

	// Execution defines shared Terraform/OpenTofu execution semantics.
	Execution ExecutionConfig `yaml:"execution,omitempty" json:"execution,omitempty" jsonschema:"description=Shared execution settings for Terraform/OpenTofu"` //nolint:modernize // yaml/v4 does not support omitzero

	// Structure defines the directory structure pattern
	Structure StructureConfig `yaml:"structure" json:"structure" jsonschema:"description=Directory structure configuration"`

	// Exclude patterns for modules to ignore
	Exclude []string `yaml:"exclude,omitempty" json:"exclude,omitempty" jsonschema:"description=Glob patterns for modules to exclude"`

	// Include patterns (if set, only matching modules are included)
	Include []string `yaml:"include,omitempty" json:"include,omitempty" jsonschema:"description=Glob patterns for modules to include (if empty, all modules are included after excludes)"`

	// LibraryModules configuration for shared/reusable modules
	LibraryModules *LibraryModulesConfig `yaml:"library_modules,omitempty" json:"library_modules,omitempty" jsonschema:"description=Configuration for library/shared modules (non-executable modules used by other modules)"`

	// Plugins holds plugin-specific configuration.
	// Each key is a plugin's ConfigKey(), value is decoded by the plugin.
	Plugins map[string]yaml.Node `yaml:"plugins,omitempty" json:"-" jsonschema:"-"`
}

// ExecutionConfig defines shared Terraform/OpenTofu execution settings.
type ExecutionConfig struct {
	Binary      string            `yaml:"binary,omitempty" json:"binary,omitempty" jsonschema:"description=Terraform/OpenTofu binary to use,enum=terraform,enum=tofu,default=terraform"`
	InitEnabled bool              `yaml:"init_enabled,omitempty" json:"init_enabled,omitempty" jsonschema:"description=Automatically run terraform init before terraform operations,default=true"`
	PlanEnabled bool              `yaml:"plan_enabled,omitempty" json:"plan_enabled,omitempty" jsonschema:"description=Enable terraform plan jobs,default=true"`
	PlanMode    string            `yaml:"plan_mode,omitempty" json:"plan_mode,omitempty" jsonschema:"description=Controls plan artifact verbosity. standard writes only plan.tfplan; detailed also writes plan.txt and plan.json for report and comment flows,enum=standard,enum=detailed,default=standard"`
	Parallelism int               `yaml:"parallelism,omitempty" json:"parallelism,omitempty" jsonschema:"description=Maximum parallel jobs for local execution,minimum=1,default=4"`
	Env         map[string]string `yaml:"env,omitempty" json:"env,omitempty" jsonschema:"description=Execution-wide environment variables"`
}

// LibraryModulesConfig defines configuration for library/shared modules
type LibraryModulesConfig struct {
	// Paths is a list of directories containing library modules (relative to root)
	// e.g., ["_modules", "shared/modules"]
	Paths []string `yaml:"paths" json:"paths" jsonschema:"description=List of directories containing library modules (relative to root)"`
}

// StructureConfig defines the directory structure
type StructureConfig struct {
	// Pattern like "{service}/{environment}/{region}/{module}"
	Pattern string `yaml:"pattern" json:"pattern" jsonschema:"description=Pattern describing module directory layout. Supported placeholders: {service}\\, {environment}\\, {region}\\, {module},default={service}/{environment}/{region}/{module}"`
	// Segments is the parsed pattern segments (derived from Pattern, not serialized)
	Segments PatternSegments `yaml:"-" json:"-"`
}
