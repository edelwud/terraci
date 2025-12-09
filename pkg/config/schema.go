package config

import (
	"encoding/json"

	"github.com/invopop/jsonschema"
)

// GenerateJSONSchema returns the JSON Schema for .terraci.yaml configuration
func GenerateJSONSchema() string {
	r := &jsonschema.Reflector{
		DoNotReference:             true,
		ExpandedStruct:             true,
		AllowAdditionalProperties:  true,
		RequiredFromJSONSchemaTags: true,
	}

	schema := r.Reflect(&Config{})
	schema.ID = "https://github.com/edelwud/terraci/raw/main/terraci.schema.json"
	schema.Title = "TerraCi Configuration"
	schema.Description = "Configuration schema for TerraCi - GitLab CI pipeline generator for Terraform monorepos"

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "{}"
	}

	return string(data)
}
