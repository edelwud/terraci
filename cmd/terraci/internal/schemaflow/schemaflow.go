// Package schemaflow builds the .terraci.yaml JSON schema from configured
// plugin schema contributors.
package schemaflow

import (
	"github.com/edelwud/terraci/pkg/config"
)

type configLoaderSource interface {
	ExtensionSchemas() map[string]any
}

// Generate returns the JSON schema for core config plus extension config
// schemas supplied by plugin config loaders.
func Generate(source configLoaderSource) string {
	var pluginSchemas map[string]any
	if source != nil {
		pluginSchemas = source.ExtensionSchemas()
	}
	return config.GenerateJSONSchema(pluginSchemas)
}
