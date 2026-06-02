package schemaflow

import (
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin"
)

type testConfigLoader struct{}

func (testConfigLoader) Name() string        { return "feature" }
func (testConfigLoader) Description() string { return "feature" }
func (testConfigLoader) ConfigKey() config.ExtensionKey {
	return config.MustExtensionKey("feature")
}
func (testConfigLoader) SchemaConfig() any {
	return &struct {
		Enabled bool `json:"enabled" yaml:"enabled"`
	}{}
}
func (testConfigLoader) DecodeAndSet(config.ExtensionDocument) error {
	return nil
}
func (testConfigLoader) IsConfigured() bool { return true }
func (testConfigLoader) IsEnabled() bool    { return true }

type testConfigSource []plugin.ConfigLoader

func (s testConfigSource) ConfigLoaders() []plugin.ConfigLoader { return []plugin.ConfigLoader(s) }

func TestGenerateIncludesExtensionSchemas(t *testing.T) {
	t.Parallel()

	schema := Generate(testConfigSource{testConfigLoader{}})
	if !strings.Contains(schema, `"feature"`) {
		t.Fatalf("schema = %s, want feature extension", schema)
	}
}
