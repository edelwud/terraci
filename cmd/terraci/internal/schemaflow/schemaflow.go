// Package schemaflow builds the .terraci.yaml JSON schema from configured
// plugin schema contributors.
package schemaflow

import (
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin"
)

type configLoaderSource interface {
	ConfigLoaders() []plugin.ConfigLoader
}

// Generate returns the JSON schema for core config plus extension config
// schemas supplied by plugin config loaders.
func Generate(source configLoaderSource) string {
	pluginSchemas := make(map[string]any)
	if source != nil {
		for _, cl := range source.ConfigLoaders() {
			pluginSchemas[cl.ConfigKey().String()] = cl.SchemaConfig()
		}
	}
	return config.GenerateJSONSchema(pluginSchemas)
}
