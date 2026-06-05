package config

import (
	"strings"
	"testing"
)

type schemaTestConfig struct {
	Enabled bool     `json:"enabled" yaml:"enabled"`
	Labels  []string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

func TestExtensionDefinitionValidation(t *testing.T) {
	t.Parallel()

	if _, err := NewExtensionDefinition(ExtensionKey{}, &schemaTestConfig{}); err == nil {
		t.Fatal("NewExtensionDefinition() error = nil, want missing key error")
	}

	var nilConfig *schemaTestConfig
	if _, err := NewExtensionDefinition(MustExtensionKey("feature"), nilConfig); err == nil {
		t.Fatal("NewExtensionDefinition() error = nil, want nil sample error")
	}

	definition, err := NewExtensionDefinition(MustExtensionKey("feature"), &schemaTestConfig{})
	if err != nil {
		t.Fatalf("NewExtensionDefinition() error = %v", err)
	}
	if got := definition.Key().String(); got != "feature" {
		t.Fatalf("Key() = %q, want feature", got)
	}
}

func TestExtensionDefinitionSetValidationAndOrdering(t *testing.T) {
	t.Parallel()

	zeta := mustExtensionDefinition(t, "zeta")
	alpha := mustExtensionDefinition(t, "alpha")

	set, err := NewExtensionDefinitionSet(zeta, alpha)
	if err != nil {
		t.Fatalf("NewExtensionDefinitionSet() error = %v", err)
	}
	definitions := set.Definitions()
	if got, want := strings.Join(extensionDefinitionKeys(definitions), ","), "alpha,zeta"; got != want {
		t.Fatalf("definition keys = %s, want %s", got, want)
	}

	if _, err := NewExtensionDefinitionSet(alpha, alpha); err == nil {
		t.Fatal("NewExtensionDefinitionSet() error = nil, want duplicate key error")
	}
}

func TestGenerateJSONSchemaIncludesExtensionDefinitions(t *testing.T) {
	t.Parallel()

	set, err := NewExtensionDefinitionSet(mustExtensionDefinition(t, "feature"))
	if err != nil {
		t.Fatalf("NewExtensionDefinitionSet() error = %v", err)
	}
	schema, err := GenerateJSONSchema(set)
	if err != nil {
		t.Fatalf("GenerateJSONSchema() error = %v", err)
	}
	for _, want := range []string{`"extensions"`, `"feature"`, `"enabled"`, `"labels"`} {
		if !strings.Contains(schema, want) {
			t.Fatalf("schema missing %s: %s", want, schema)
		}
	}
}

func mustExtensionDefinition(t *testing.T, key string) ExtensionDefinition {
	t.Helper()
	definition, err := NewExtensionDefinition(MustExtensionKey(key), &schemaTestConfig{})
	if err != nil {
		t.Fatalf("NewExtensionDefinition(%q) error = %v", key, err)
	}
	return definition
}

func extensionDefinitionKeys(definitions []ExtensionDefinition) []string {
	keys := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		keys = append(keys, definition.Key().String())
	}
	return keys
}
