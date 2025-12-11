// Package e2e provides end-to-end tests for terraci pipeline generation
package e2e

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/graph"
	"github.com/edelwud/terraci/internal/parser"
	"github.com/edelwud/terraci/internal/pipeline/gitlab"
	"github.com/edelwud/terraci/pkg/config"
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
	Name        string
	Dir         string
	Config      *config.Config
	Modules     []*discovery.Module
	ModuleIndex *discovery.ModuleIndex
	DepGraph    *graph.DependencyGraph
	Generator   *gitlab.Generator
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

	// Scan modules
	scanner := discovery.NewScanner(dir)
	scanner.MinDepth = cfg.Structure.MinDepth
	scanner.MaxDepth = cfg.Structure.MaxDepth

	modules, err := scanner.Scan()
	if err != nil {
		t.Fatalf("failed to scan modules for fixture %s: %v", name, err)
	}

	if len(modules) == 0 {
		t.Fatalf("no modules found in fixture %s", name)
	}

	// Build module index
	moduleIndex := discovery.NewModuleIndex(modules)

	// Parse dependencies
	hclParser := parser.NewParser()
	depExtractor := parser.NewDependencyExtractor(hclParser, moduleIndex)
	deps, _ := depExtractor.ExtractAllDependencies()

	// Build dependency graph
	depGraph := graph.BuildFromDependencies(modules, deps)

	// Create generator
	generator := gitlab.NewGenerator(cfg, depGraph, modules)

	return &Fixture{
		Name:        name,
		Dir:         dir,
		Config:      cfg,
		Modules:     modules,
		ModuleIndex: moduleIndex,
		DepGraph:    depGraph,
		Generator:   generator,
	}
}

// LoadFixtureWithConfig loads a fixture and applies custom config modifications
func LoadFixtureWithConfig(t *testing.T, name string, modifyConfig func(*config.Config)) *Fixture {
	t.Helper()

	fixture := LoadFixture(t, name)
	modifyConfig(fixture.Config)

	// Recreate generator with modified config
	fixture.Generator = gitlab.NewGenerator(fixture.Config, fixture.DepGraph, fixture.Modules)

	return fixture
}

// GetModuleByName finds a module by its module name (e.g., "vpc", "eks")
func (f *Fixture) GetModuleByName(moduleName string) *discovery.Module {
	for _, m := range f.Modules {
		if m.Module == moduleName || m.Name() == moduleName {
			return m
		}
	}
	return nil
}

// GetModulesByEnvironment returns all modules for a specific environment
func (f *Fixture) GetModulesByEnvironment(env string) []*discovery.Module {
	var result []*discovery.Module
	for _, m := range f.Modules {
		if m.Environment == env {
			result = append(result, m)
		}
	}
	return result
}

// GetModulesByService returns all modules for a specific service
func (f *Fixture) GetModulesByService(service string) []*discovery.Module {
	var result []*discovery.Module
	for _, m := range f.Modules {
		if m.Service == service {
			result = append(result, m)
		}
	}
	return result
}

// AssertJobExists checks that a job with the given name exists in the pipeline
func AssertJobExists(t *testing.T, pipeline *gitlab.Pipeline, jobName string) {
	t.Helper()
	if _, exists := pipeline.Jobs[jobName]; !exists {
		t.Errorf("expected job %s to exist", jobName)
	}
}

// AssertJobNotExists checks that a job with the given name does not exist
func AssertJobNotExists(t *testing.T, pipeline *gitlab.Pipeline, jobName string) {
	t.Helper()
	if _, exists := pipeline.Jobs[jobName]; exists {
		t.Errorf("expected job %s to not exist", jobName)
	}
}

// AssertStageExists checks that a stage with the given name exists in the pipeline
func AssertStageExists(t *testing.T, pipeline *gitlab.Pipeline, stageName string) {
	t.Helper()
	for _, stage := range pipeline.Stages {
		if stage == stageName {
			return
		}
	}
	t.Errorf("expected stage %s to exist", stageName)
}

// AssertStageNotExists checks that a stage does not exist
func AssertStageNotExists(t *testing.T, pipeline *gitlab.Pipeline, stageName string) {
	t.Helper()
	for _, stage := range pipeline.Stages {
		if stage == stageName {
			t.Errorf("expected stage %s to not exist", stageName)
			return
		}
	}
}

// AssertJobHasNeed checks that a job depends on another job
func AssertJobHasNeed(t *testing.T, pipeline *gitlab.Pipeline, jobName, needJob string) {
	t.Helper()
	job, exists := pipeline.Jobs[jobName]
	if !exists {
		t.Errorf("job %s does not exist", jobName)
		return
	}

	for _, need := range job.Needs {
		if need.Job == needJob {
			return
		}
	}
	t.Errorf("job %s should depend on %s", jobName, needJob)
}

// AssertJobNotHasNeed checks that a job does not depend on another job
func AssertJobNotHasNeed(t *testing.T, pipeline *gitlab.Pipeline, jobName, needJob string) {
	t.Helper()
	job, exists := pipeline.Jobs[jobName]
	if !exists {
		t.Errorf("job %s does not exist", jobName)
		return
	}

	for _, need := range job.Needs {
		if need.Job == needJob {
			t.Errorf("job %s should not depend on %s", jobName, needJob)
			return
		}
	}
}

// AssertJobCount checks the total number of jobs in the pipeline
func AssertJobCount(t *testing.T, pipeline *gitlab.Pipeline, expected int) {
	t.Helper()
	if len(pipeline.Jobs) != expected {
		t.Errorf("expected %d jobs, got %d", expected, len(pipeline.Jobs))
	}
}

// AssertStageCount checks the total number of stages in the pipeline
func AssertStageCount(t *testing.T, pipeline *gitlab.Pipeline, expected int) {
	t.Helper()
	if len(pipeline.Stages) != expected {
		t.Errorf("expected %d stages, got %d", expected, len(pipeline.Stages))
	}
}

// CountJobsByPrefix counts jobs that start with a given prefix
func CountJobsByPrefix(pipeline *gitlab.Pipeline, prefix string) int {
	count := 0
	for jobName := range pipeline.Jobs {
		if len(jobName) >= len(prefix) && jobName[:len(prefix)] == prefix {
			count++
		}
	}
	return count
}

// GetJobNeeds returns all job names that a job depends on
func GetJobNeeds(pipeline *gitlab.Pipeline, jobName string) []string {
	job, exists := pipeline.Jobs[jobName]
	if !exists {
		return nil
	}

	needs := make([]string, 0, len(job.Needs))
	for _, need := range job.Needs {
		needs = append(needs, need.Job)
	}
	return needs
}
