package schemaflow

import (
	"strings"
	"testing"
)

type testConfigSource map[string]any

func (s testConfigSource) ExtensionSchemas() map[string]any { return map[string]any(s) }

func TestGenerateIncludesExtensionSchemas(t *testing.T) {
	t.Parallel()

	schema := Generate(testConfigSource{
		"feature": &struct {
			Enabled bool `json:"enabled" yaml:"enabled"`
		}{},
	})
	if !strings.Contains(schema, `"feature"`) {
		t.Fatalf("schema = %s, want feature extension", schema)
	}
}
