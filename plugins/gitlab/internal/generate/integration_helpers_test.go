package generate

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/parser"
	"github.com/edelwud/terraci/pkg/pipeline"
)

// testdataDir returns the absolute path to the testdata directory
func testdataDir(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get caller info")
	}
	return filepath.Join(filepath.Dir(filename), "..", "testdata")
}

// fixtureDir returns the absolute path to a specific fixture directory
func fixtureDir(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(testdataDir(t), name)
}

// Fixture represents a loaded test fixture with all components
type Fixture struct {
	Name          string
	Dir           string
	Config        *config.Config
	GLConfig      *Config
	Contributions []*pipeline.Contribution
	Modules       []*discovery.Module
	ModuleIndex   *discovery.ModuleIndex
	DepGraph      *graph.DependencyGraph
	Generator     *Generator
}

// decodeGLConfig extracts the gitlab plugin config from the plugins map.
func decodeGLConfig(cfg *config.Config) *Config {
	glCfg := &Config{
		TerraformBinary: "terraform",
		Image:           Image{Name: "hashicorp/terraform:1.6"},
		PlanEnabled:     true,
		InitEnabled:     true,
	}
	if err := cfg.PluginConfig("gitlab", glCfg); err != nil {
		return glCfg
	}
	return glCfg
}

// LoadFixture loads a complete test fixture by name
func LoadFixture(t *testing.T, name string) *Fixture {
	t.Helper()

	dir := fixtureDir(t, name)

	// Load config
	cfg, err := config.LoadOrDefault(dir)
	if err != nil {
		t.Fatalf("failed to load config for fixture %s: %v", name, err)
	}

	glCfg := decodeGLConfig(cfg)

	// Scan modules
	scanner := discovery.NewScanner(dir, cfg.Structure.Segments)

	modules, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("failed to scan modules for fixture %s: %v", name, err)
	}

	if len(modules) == 0 {
		t.Fatalf("no modules found in fixture %s", name)
	}

	// Build module index
	moduleIndex := discovery.NewModuleIndex(modules)

	// Parse dependencies
	hclParser := parser.NewParser(cfg.Structure.Segments)
	depExtractor := parser.NewDependencyExtractor(hclParser, moduleIndex)
	deps, _ := depExtractor.ExtractAllDependencies(context.Background())

	// Build dependency graph
	depGraph := graph.BuildFromDependencies(modules, deps)

	// Create generator (no contributions in test fixtures)
	generator := NewGenerator(glCfg, nil, depGraph, modules)

	return &Fixture{
		Name:        name,
		Dir:         dir,
		Config:      cfg,
		GLConfig:    glCfg,
		Modules:     modules,
		ModuleIndex: moduleIndex,
		DepGraph:    depGraph,
		Generator:   generator,
	}
}

// GetModuleByName finds a module by its module name (e.g., "vpc", "eks")
func (f *Fixture) GetModuleByName(moduleName string) *discovery.Module {
	for _, m := range f.Modules {
		if m.Get("module") == moduleName || m.Name() == moduleName {
			return m
		}
	}
	return nil
}

// GetModulesByEnvironment returns all modules for a specific environment
func (f *Fixture) GetModulesByEnvironment(env string) []*discovery.Module {
	var result []*discovery.Module
	for _, m := range f.Modules {
		if m.Get("environment") == env {
			result = append(result, m)
		}
	}
	return result
}
