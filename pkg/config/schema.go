package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"

	"github.com/invopop/jsonschema"
)

type configSchema struct {
	ServiceDir     string                      `json:"service_dir,omitempty" jsonschema:"description=Service directory for cache and artifacts,default=.terraci"`
	Execution      executionSchema             `json:"execution,omitempty" jsonschema:"description=Shared execution settings for Terraform/OpenTofu"` //nolint:modernize // jsonschema reflector expects omitempty on nested schema DTOs.
	Structure      structureSchema             `json:"structure" jsonschema:"description=Directory structure configuration"`
	Exclude        []string                    `json:"exclude,omitempty" jsonschema:"description=Glob patterns for modules to exclude"`
	Include        []string                    `json:"include,omitempty" jsonschema:"description=Glob patterns for modules to include (if empty, all modules are included after excludes)"`
	LibraryModules *libraryModulesConfigSchema `json:"library_modules,omitempty" jsonschema:"description=Configuration for library/shared modules (non-executable modules used by other modules)"`
}

type executionSchema struct {
	Binary      string            `json:"binary,omitempty" jsonschema:"description=Terraform/OpenTofu binary to use,enum=terraform,enum=tofu,default=terraform"`
	InitEnabled bool              `json:"init_enabled,omitempty" jsonschema:"description=Automatically run terraform init before terraform operations,default=true"`
	Parallelism int               `json:"parallelism,omitempty" jsonschema:"description=Maximum parallel jobs for local execution,minimum=1,default=4"`
	Env         map[string]string `json:"env,omitempty" jsonschema:"description=Execution-wide environment variables"`
}

type structureSchema struct {
	Pattern string `json:"pattern" jsonschema:"description=Pattern describing module directory layout. Supported placeholders: {service}\\, {environment}\\, {region}\\, {module},default={service}/{environment}/{region}/{module}"`
}

type libraryModulesConfigSchema struct {
	Paths []string `json:"paths" jsonschema:"description=List of directories containing library modules (relative to root)"`
}

// ExtensionDefinition describes one typed extension config section for schema
// generation.
type ExtensionDefinition struct {
	key        ExtensionKey
	sampleType reflect.Type
}

// ExtensionDefinitionSet is a duplicate-free, deterministic set of extension
// config schema definitions.
type ExtensionDefinitionSet struct {
	definitions []ExtensionDefinition
}

// NewExtensionDefinition validates and captures one extension config shape for
// JSON schema generation.
func NewExtensionDefinition[C any](key ExtensionKey, sample C) (ExtensionDefinition, error) {
	if key.String() == "" {
		return ExtensionDefinition{}, errors.New("extension definition key is required")
	}
	sampleType := reflect.TypeOf(sample)
	if sampleType == nil {
		return ExtensionDefinition{}, fmt.Errorf("extension %q schema sample is nil", key.String())
	}
	if isNilValue(sample) {
		return ExtensionDefinition{}, fmt.Errorf("extension %q schema sample is nil", key.String())
	}
	return ExtensionDefinition{
		key:        key,
		sampleType: sampleType,
	}, nil
}

// Key returns the validated extension key.
func (d ExtensionDefinition) Key() ExtensionKey {
	return d.key
}

func (d ExtensionDefinition) reflectType() reflect.Type {
	return d.sampleType
}

// NewExtensionDefinitionSet builds a duplicate-free definition set sorted by
// extension key.
func NewExtensionDefinitionSet(defs ...ExtensionDefinition) (ExtensionDefinitionSet, error) {
	seen := make(map[string]struct{}, len(defs))
	definitions := make([]ExtensionDefinition, 0, len(defs))
	for i := range defs {
		def := defs[i]
		key := def.key.String()
		if key == "" {
			return ExtensionDefinitionSet{}, errors.New("extension definition key is required")
		}
		if def.sampleType == nil {
			return ExtensionDefinitionSet{}, fmt.Errorf("extension %q schema sample is nil", key)
		}
		if _, exists := seen[key]; exists {
			return ExtensionDefinitionSet{}, fmt.Errorf("duplicate extension definition %q", key)
		}
		seen[key] = struct{}{}
		definitions = append(definitions, def)
	}
	sort.Slice(definitions, func(i, j int) bool {
		return definitions[i].key.String() < definitions[j].key.String()
	})
	return ExtensionDefinitionSet{definitions: definitions}, nil
}

// Definitions returns defensive definition value copies in deterministic order.
func (s ExtensionDefinitionSet) Definitions() []ExtensionDefinition {
	if len(s.definitions) == 0 {
		return nil
	}
	return append([]ExtensionDefinition(nil), s.definitions...)
}

// Len returns the number of extension definitions.
func (s ExtensionDefinitionSet) Len() int {
	return len(s.definitions)
}

func isNilValue(sample any) bool {
	value := reflect.ValueOf(sample)
	if !value.IsValid() {
		return true
	}
	kind := value.Kind()
	nilCapable := kind == reflect.Chan ||
		kind == reflect.Func ||
		kind == reflect.Interface ||
		kind == reflect.Map ||
		kind == reflect.Pointer ||
		kind == reflect.Slice
	return nilCapable && value.IsNil()
}

// GenerateJSONSchema returns the JSON Schema for .terraci.yaml configuration.
func GenerateJSONSchema(definitions ExtensionDefinitionSet) (string, error) {
	r := &jsonschema.Reflector{
		DoNotReference:             true,
		ExpandedStruct:             true,
		AllowAdditionalProperties:  true,
		RequiredFromJSONSchemaTags: true,
	}

	schema := r.Reflect(&configSchema{})
	schema.ID = "https://github.com/edelwud/terraci/raw/main/terraci.schema.json"
	schema.Title = "TerraCi Configuration"
	schema.Description = "Configuration schema for TerraCi - CI pipeline generator for Terraform monorepos"

	if definitions.Len() > 0 {
		extensionsProp := &jsonschema.Schema{
			Type:                 "object",
			Description:          "Extension-specific configuration sections",
			Properties:           jsonschema.NewProperties(),
			AdditionalProperties: jsonschema.TrueSchema,
		}
		for _, def := range definitions.Definitions() {
			subSchema := r.ReflectFromType(def.reflectType())
			subSchema.Version = ""
			subSchema.ID = ""
			extensionsProp.Properties.Set(def.Key().String(), subSchema)
		}
		schema.Properties.Set("extensions", extensionsProp)
	}

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal JSON schema: %w", err)
	}

	return string(data), nil
}
