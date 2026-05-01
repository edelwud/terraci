package config

import (
	"encoding/json"
	"reflect"

	"github.com/invopop/jsonschema"
)

// GenerateJSONSchema returns the JSON Schema for .terraci.yaml configuration.
// extensionSchemas maps each extension key to a sample value whose Go type
// shape will be reflected as the section's sub-schema.
func GenerateJSONSchema(extensionSchemas map[string]any) string {
	r := &jsonschema.Reflector{
		DoNotReference:             true,
		ExpandedStruct:             true,
		AllowAdditionalProperties:  true,
		RequiredFromJSONSchemaTags: true,
	}

	schema := r.Reflect(&Config{})
	schema.ID = "https://github.com/edelwud/terraci/raw/main/terraci.schema.json"
	schema.Title = "TerraCi Configuration"
	schema.Description = "Configuration schema for TerraCi - CI pipeline generator for Terraform monorepos"

	if len(extensionSchemas) > 0 {
		extensionsProp := &jsonschema.Schema{
			Type:                 "object",
			Description:          "Extension-specific configuration sections",
			Properties:           jsonschema.NewProperties(),
			AdditionalProperties: jsonschema.TrueSchema,
		}
		for key, cfg := range extensionSchemas {
			subSchema := r.ReflectFromType(reflect.TypeOf(cfg))
			subSchema.Version = ""
			subSchema.ID = ""
			extensionsProp.Properties.Set(key, subSchema)
		}
		schema.Properties.Set("extensions", extensionsProp)
	}

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "{}"
	}

	return string(data)
}
