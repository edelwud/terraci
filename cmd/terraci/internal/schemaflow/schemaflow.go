// Package schemaflow builds the .terraci.yaml JSON schema from configured
// plugin schema contributors.
package schemaflow

import (
	"github.com/edelwud/terraci/pkg/config"
)

type configLoaderSource interface {
	ExtensionDefinitions() (config.ExtensionDefinitionSet, error)
}

// Generate returns the JSON schema for core config plus extension config
// schemas supplied by plugin config loaders.
func Generate(source configLoaderSource) (string, error) {
	var definitions config.ExtensionDefinitionSet
	if source != nil {
		var err error
		definitions, err = source.ExtensionDefinitions()
		if err != nil {
			return "", err
		}
	}
	return config.GenerateJSONSchema(definitions)
}
