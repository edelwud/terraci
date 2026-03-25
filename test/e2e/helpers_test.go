// Package e2e provides end-to-end tests for terraci pipeline generation
package e2e

import (
	"context"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/parser"
	"github.com/edelwud/terraci/pkg/pipeline"
	glplugin "github.com/edelwud/terraci/plugins/gitlab"
)

// testdataDir returns the absolute path to the testdata directory
func testdataDir(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get caller info")
	}
	return filepath.Join(filepath.Dir(filename), "testdata")
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
	GLConfig      *glplugin.Config
	Contributions []*pipeline.Contribution
	Modules       []*discovery.Module
	ModuleIndex   *discovery.ModuleIndex
	DepGraph      *graph.DependencyGraph
	Generator     *glplugin.Generator
}

// decodeGLConfig extracts the gitlab plugin config from the plugins map.
func decodeGLConfig(cfg *config.Config) *glplugin.Config {
	glCfg := &glplugin.Config{
		TerraformBinary: "terraform",
		Image:           glplugin.Image{Name: "hashicorp/terraform:1.6"},
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
	generator := glplugin.NewGenerator(glCfg, nil, depGraph, modules)

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

// LoadFixtureWithConfig loads a fixture and applies custom config modifications.
// The modifyConfig callback receives the Config directly.
func LoadFixtureWithConfig(t *testing.T, name string, modifyConfig func(*glplugin.Config)) *Fixture {
	t.Helper()

	fixture := LoadFixture(t, name)
	modifyConfig(fixture.GLConfig)

	// Recreate generator with modified config
	fixture.Generator = glplugin.NewGenerator(fixture.GLConfig, fixture.Contributions, fixture.DepGraph, fixture.Modules)

	return fixture
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

// GenerateAndValidate generates a pipeline and runs basic validation
func (f *Fixture) GenerateAndValidate(t *testing.T) *glplugin.Pipeline {
	t.Helper()

	result, err := f.Generator.Generate(f.Modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	pl, ok := result.(*glplugin.Pipeline)
	if !ok {
		t.Fatal("expected *glplugin.Pipeline type")
	}

	return pl
}

// ModuleIDs returns the IDs of all modules sorted
func (f *Fixture) ModuleIDs() []string {
	ids := make([]string, len(f.Modules))
	for i, m := range f.Modules {
		ids[i] = m.ID()
	}
	slices.Sort(ids)
	return ids
}

// GetModulesByService returns all modules for a specific service
func (f *Fixture) GetModulesByService(service string) []*discovery.Module {
	var result []*discovery.Module
	for _, m := range f.Modules {
		if m.Get("service") == service {
			result = append(result, m)
		}
	}
	return result
}

// --- Assertion helpers ---

// AssertJobExists checks that a job exists in the pipeline
func AssertJobExists(t *testing.T, pl *glplugin.Pipeline, jobName string) {
	t.Helper()
	if _, ok := pl.Jobs[jobName]; !ok {
		t.Errorf("expected job %q to exist", jobName)
	}
}

// AssertJobNotExists checks that a job does NOT exist in the pipeline
func AssertJobNotExists(t *testing.T, pl *glplugin.Pipeline, jobName string) {
	t.Helper()
	if _, ok := pl.Jobs[jobName]; ok {
		t.Errorf("expected job %q to NOT exist", jobName)
	}
}

// AssertStageExists checks that a stage exists in the pipeline
func AssertStageExists(t *testing.T, pl *glplugin.Pipeline, stageName string) {
	t.Helper()
	if !slices.Contains(pl.Stages, stageName) {
		t.Errorf("expected stage %q to exist in %v", stageName, pl.Stages)
	}
}

// AssertStageNotExists checks that a stage does NOT exist in the pipeline
func AssertStageNotExists(t *testing.T, pl *glplugin.Pipeline, stageName string) {
	t.Helper()
	if slices.Contains(pl.Stages, stageName) {
		t.Errorf("expected stage %q to NOT exist in %v", stageName, pl.Stages)
	}
}

// AssertJobHasNeed checks that a job has a specific dependency
func AssertJobHasNeed(t *testing.T, pl *glplugin.Pipeline, jobName, needJob string) {
	t.Helper()
	job, ok := pl.Jobs[jobName]
	if !ok {
		t.Fatalf("job %q not found", jobName)
	}
	for _, need := range job.Needs {
		if need.Job == needJob {
			return
		}
	}
	t.Errorf("job %q should need %q", jobName, needJob)
}

// AssertJobNotHasNeed checks that a job does NOT have a specific dependency
func AssertJobNotHasNeed(t *testing.T, pl *glplugin.Pipeline, jobName, needJob string) {
	t.Helper()
	job, ok := pl.Jobs[jobName]
	if !ok {
		return // job doesn't exist, so it can't have the need
	}
	for _, need := range job.Needs {
		if need.Job == needJob {
			t.Errorf("job %q should NOT need %q", jobName, needJob)
			return
		}
	}
}

// AssertJobCount checks the number of jobs in the pipeline
func AssertJobCount(t *testing.T, pl *glplugin.Pipeline, expected int) {
	t.Helper()
	if len(pl.Jobs) != expected {
		t.Errorf("expected %d jobs, got %d", expected, len(pl.Jobs))
	}
}

// AssertStageCount checks the number of stages in the pipeline
func AssertStageCount(t *testing.T, pl *glplugin.Pipeline, expected int) {
	t.Helper()
	if len(pl.Stages) != expected {
		t.Errorf("expected %d stages, got %d: %v", expected, len(pl.Stages), pl.Stages)
	}
}

// CountJobsByPrefix counts jobs whose name starts with the given prefix
func CountJobsByPrefix(pl *glplugin.Pipeline, prefix string) int {
	count := 0
	for name := range pl.Jobs {
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			count++
		}
	}
	return count
}

// GetJobNeeds returns the names of jobs that the given job depends on
func GetJobNeeds(pl *glplugin.Pipeline, jobName string) []string {
	job, ok := pl.Jobs[jobName]
	if !ok {
		return nil
	}
	needs := make([]string, len(job.Needs))
	for i, need := range job.Needs {
		needs[i] = need.Job
	}
	return needs
}
