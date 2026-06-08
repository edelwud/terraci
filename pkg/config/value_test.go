package config

import (
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestConfigGettersReturnDefensiveCopies(t *testing.T) {
	t.Parallel()

	initEnabled := false
	execution, err := NewExecutionConfig(ExecutionConfigOptions{
		Binary:      ExecutionBinaryTofu,
		InitEnabled: &initEnabled,
		Parallelism: 8,
		Env:         map[string]string{"TF_LOG": "info"},
	})
	if err != nil {
		t.Fatalf("NewExecutionConfig() error = %v", err)
	}
	libraryModules, err := NewLibraryModulesConfig(LibraryModulesConfigOptions{Paths: []string{"_modules"}})
	if err != nil {
		t.Fatalf("NewLibraryModulesConfig() error = %v", err)
	}
	cfg := Default()
	cfg.execution = execution
	cfg.exclude = []string{"old-exclude"}
	cfg.include = []string{"old-include"}
	cfg.libraryModules = &libraryModules

	env := cfg.Execution().Env()
	env["TF_LOG"] = "trace"
	if got := cfg.Execution().Env()["TF_LOG"]; got != "info" {
		t.Fatalf("Execution().Env() leaked mutation: %q", got)
	}

	exclude := cfg.Exclude()
	exclude[0] = "changed"
	if got := cfg.Exclude()[0]; got != "old-exclude" {
		t.Fatalf("Exclude() leaked mutation: %q", got)
	}

	include := cfg.Include()
	include[0] = "changed"
	if got := cfg.Include()[0]; got != "old-include" {
		t.Fatalf("Include() leaked mutation: %q", got)
	}

	libraries := cfg.LibraryModules()
	libraries.paths[0] = "changed"
	if got := cfg.LibraryModules().Paths()[0]; got != "_modules" {
		t.Fatalf("LibraryModules() leaked mutation: %q", got)
	}
}

func TestConfigExtensionDocumentsAreDefensive(t *testing.T) {
	t.Parallel()

	type extensionConfig struct {
		Enabled bool `yaml:"enabled"`
	}
	var node yaml.Node
	if err := yaml.Unmarshal([]byte("enabled: true\n"), &node); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
	cfg := Default()
	cfg.extensions["feature"] = node

	doc, ok := cfg.Extension(MustExtensionKey("feature"))
	if !ok {
		t.Fatal("Extension(feature) missing")
	}
	cfg.extensions["feature"].Content[0].Content[1].Value = "false"

	var got extensionConfig
	if err := doc.Decode(&got); err != nil {
		t.Fatalf("Extension(feature).Decode() error = %v", err)
	}
	if !got.Enabled {
		t.Fatal("Extension() decoded mutated source node, want captured true")
	}
}
