package generate

import (
	"testing"

	"github.com/edelwud/terraci/pkg/ci/citest"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/pipeline"
)

type generatorScenario struct {
	t             *testing.T
	cfg           *testCfg
	modules       []*discovery.Module
	dependencies  map[string][]string
	targetModules []*discovery.Module
}

func newGeneratorScenario(t *testing.T) *generatorScenario {
	t.Helper()
	return &generatorScenario{
		t:   t,
		cfg: createTestConfig(),
	}
}

func (s *generatorScenario) withConfig(apply func(*Config)) *generatorScenario {
	s.t.Helper()
	apply(s.cfg.GitLab)
	return s
}

func (s *generatorScenario) withExecution(apply func(*execution.Config)) *generatorScenario {
	s.t.Helper()
	apply(&s.cfg.Execution)
	return s
}

func (s *generatorScenario) withModules(modules ...*discovery.Module) *generatorScenario {
	s.t.Helper()
	s.modules = modules
	return s
}

func (s *generatorScenario) withDependencies(deps map[string][]string) *generatorScenario {
	s.t.Helper()
	s.dependencies = deps
	return s
}

func (s *generatorScenario) withTargets(targets ...*discovery.Module) *generatorScenario {
	s.t.Helper()
	s.targetModules = targets
	return s
}

func (s *generatorScenario) generator() *Generator {
	s.t.Helper()
	depGraph := citest.DependencyGraph(s.modules, s.dependencies)
	return NewGenerator(s.cfg.GitLab, s.cfg.Execution, s.cfg.Contributions, depGraph, s.modules)
}

func (s *generatorScenario) generate() *Pipeline {
	s.t.Helper()
	genPipeline, err := s.generator().Generate(s.generateTargets())
	if err != nil {
		s.t.Fatalf("Generate failed: %v", err)
	}

	gitlabPipeline, ok := genPipeline.(*Pipeline)
	if !ok {
		s.t.Fatal("expected *Pipeline type")
	}

	return gitlabPipeline
}

func (s *generatorScenario) dryRun() *pipeline.DryRunResult {
	s.t.Helper()
	result, err := s.generator().DryRun(s.generateTargets())
	if err != nil {
		s.t.Fatalf("DryRun failed: %v", err)
	}
	return result
}

func (s *generatorScenario) generateTargets() []*discovery.Module {
	if s.targetModules != nil {
		return s.targetModules
	}
	return s.modules
}

func mustJob(t *testing.T, gitlabPipeline *Pipeline, name string) *Job {
	t.Helper()
	job := gitlabPipeline.Jobs[name]
	if job == nil {
		t.Fatalf("job %q not found", name)
	}
	return job
}
