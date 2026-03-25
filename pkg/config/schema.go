package config

import (
	"encoding/json"
	"reflect"

	"github.com/invopop/jsonschema"
)

// GenerateJSONSchema returns the JSON Schema for .terraci.yaml configuration
func GenerateJSONSchema(pluginSchemas map[string]any) string {
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

	// Add plugins property with sub-schemas from registered plugins
	if len(pluginSchemas) > 0 {
		pluginsProp := &jsonschema.Schema{
			Type:                 "object",
			Description:          "Plugin-specific configuration",
			Properties:           jsonschema.NewProperties(),
			AdditionalProperties: jsonschema.TrueSchema,
		}
		for key, cfg := range pluginSchemas {
			subSchema := r.ReflectFromType(reflect.TypeOf(cfg))
			// Remove the $schema and $id from sub-schemas
			subSchema.Version = ""
			subSchema.ID = ""
			pluginsProp.Properties.Set(key, subSchema)
		}
		schema.Properties.Set("plugins", pluginsProp)
	}

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "{}"
	}

	return string(data)
}
