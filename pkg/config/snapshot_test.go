package config

import (
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestConfigCloneDeepCopiesMutableFields(t *testing.T) {
	t.Parallel()

	var node yaml.Node
	if err := yaml.Unmarshal([]byte("enabled: true\nlabels: [one]\n"), &node); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
	cfg := &Config{
		ServiceDir: DefaultServiceDir,
		Execution: ExecutionConfig{
			Binary: "terraform",
			Env:    map[string]string{"TF_LOG": "info"},
		},
		Structure: StructureConfig{
			Pattern:  "{service}/{module}",
			Segments: PatternSegments{"service", "module"},
		},
		Exclude:        []string{"old-exclude"},
		Include:        []string{"old-include"},
		LibraryModules: &LibraryModulesConfig{Paths: []string{"_modules"}},
		Extensions:     map[string]yaml.Node{"summary": node},
	}

	clone := cfg.Clone()
	cfg.Execution.Env["TF_LOG"] = "trace"
	cfg.Structure.Segments[0] = "changed"
	cfg.Exclude[0] = "changed"
	cfg.Include[0] = "changed"
	cfg.LibraryModules.Paths[0] = "changed"
	cfg.Extensions["summary"].Content[0].Content[0].Value = "changed"

	if got := clone.Execution.Env["TF_LOG"]; got != "info" {
		t.Fatalf("clone Execution.Env leaked mutation: %q", got)
	}
	if got := clone.Structure.Segments[0]; got != "service" {
		t.Fatalf("clone Structure.Segments leaked mutation: %q", got)
	}
	if got := clone.Exclude[0]; got != "old-exclude" {
		t.Fatalf("clone Exclude leaked mutation: %q", got)
	}
	if got := clone.Include[0]; got != "old-include" {
		t.Fatalf("clone Include leaked mutation: %q", got)
	}
	if got := clone.LibraryModules.Paths[0]; got != "_modules" {
		t.Fatalf("clone LibraryModules leaked mutation: %q", got)
	}
	if got := clone.Extensions["summary"].Content[0].Content[0].Value; got != "enabled" {
		t.Fatalf("clone Extensions leaked mutation: %q", got)
	}
}

func TestSnapshotIsImmutableAndReturnsDefensiveCopies(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	cfg.ServiceDir = DefaultServiceDir
	cfg.Execution.Env = map[string]string{"A": "B"}
	cfg.Exclude = []string{"exclude"}
	cfg.Include = []string{"include"}
	cfg.LibraryModules = &LibraryModulesConfig{Paths: []string{"_modules"}}

	snapshot := cfg.Snapshot()
	cfg.ServiceDir = ".changed"
	cfg.Execution.Env["A"] = "changed"
	cfg.Exclude[0] = "changed"
	cfg.Include[0] = "changed"
	cfg.LibraryModules.Paths[0] = "changed"

	if got := snapshot.ServiceDir(); got != DefaultServiceDir {
		t.Fatalf("ServiceDir() = %q, want captured value", got)
	}
	execution := snapshot.Execution()
	execution.Env["A"] = "mutated"
	if got := snapshot.Execution().Env["A"]; got != "B" {
		t.Fatalf("Execution() leaked mutation: %q", got)
	}
	exclude := snapshot.Exclude()
	exclude[0] = "mutated"
	if got := snapshot.Exclude()[0]; got != "exclude" {
		t.Fatalf("Exclude() leaked mutation: %q", got)
	}
	include := snapshot.Include()
	include[0] = "mutated"
	if got := snapshot.Include()[0]; got != "include" {
		t.Fatalf("Include() leaked mutation: %q", got)
	}
	libraryModules := snapshot.LibraryModules()
	libraryModules.Paths[0] = "mutated"
	if got := snapshot.LibraryModules().Paths[0]; got != "_modules" {
		t.Fatalf("LibraryModules() leaked mutation: %q", got)
	}

	mutable := snapshot.MutableCopy()
	mutable.ServiceDir = ".copy"
	if got := snapshot.ServiceDir(); got != DefaultServiceDir {
		t.Fatalf("MutableCopy() leaked mutation: %q", got)
	}
}

func TestSnapshotExtensionDecodesCapturedYAML(t *testing.T) {
	t.Parallel()

	type extensionConfig struct {
		Enabled bool `yaml:"enabled"`
	}
	var node yaml.Node
	if err := yaml.Unmarshal([]byte("enabled: true\n"), &node); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
	cfg := DefaultConfig()
	cfg.Extensions["feature"] = node
	snapshot := NewSnapshot(cfg)
	cfg.Extensions["feature"].Content[0].Content[1].Value = "false"

	var got extensionConfig
	if err := snapshot.Extension("feature", &got); err != nil {
		t.Fatalf("Extension() error = %v", err)
	}
	if !got.Enabled {
		t.Fatal("Extension() decoded mutated source node, want captured true")
	}
}
