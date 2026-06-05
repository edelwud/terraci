package schemaflow

import (
	"errors"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/config"
)

type testConfigSource struct {
	definitions config.ExtensionDefinitionSet
	err         error
}

func (s testConfigSource) ExtensionDefinitions() (config.ExtensionDefinitionSet, error) {
	return s.definitions, s.err
}

func TestGenerateIncludesExtensionDefinitions(t *testing.T) {
	t.Parallel()

	definition, err := config.NewExtensionDefinition(config.MustExtensionKey("feature"), &struct {
		Enabled bool `json:"enabled" yaml:"enabled"`
	}{})
	if err != nil {
		t.Fatalf("NewExtensionDefinition() error = %v", err)
	}
	set, err := config.NewExtensionDefinitionSet(definition)
	if err != nil {
		t.Fatalf("NewExtensionDefinitionSet() error = %v", err)
	}
	schema, err := Generate(testConfigSource{definitions: set})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if !strings.Contains(schema, `"feature"`) {
		t.Fatalf("schema = %s, want feature extension", schema)
	}
}

func TestGenerateReturnsSourceError(t *testing.T) {
	t.Parallel()

	want := errors.New("definitions failed")
	if _, err := Generate(testConfigSource{err: want}); !errors.Is(err, want) {
		t.Fatalf("Generate() error = %v, want %v", err, want)
	}
}
