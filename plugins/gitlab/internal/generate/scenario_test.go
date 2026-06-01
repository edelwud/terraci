package generate

import (
	"testing"

	"github.com/edelwud/terraci/pkg/ci/citest"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/terraformrun"
)

type generatorScenario struct {
	t             *testing.T
	cfg           *testCfg
	modules       []*discovery.Module
	dependencies  map[string][]string
	targetModules []*discovery.Module
	applyEnabled  bool
}

func newGeneratorScenario(t *testing.T) *generatorScenario {
	t.Helper()
	return &generatorScenario{
		t:            t,
		cfg:          createTestConfig(),
		applyEnabled: true,
	}
}

func (s *generatorScenario) withConfig(apply func(*Config)) *generatorScenario {
	s.t.Helper()
	apply(s.cfg.GitLab)
	return s
}

func (s *generatorScenario) withExecution(apply func(*terraformrun.ProfileOptions)) *generatorScenario {
	s.t.Helper()
	opts := profileOptionsFromProfile(s.cfg.Profile)
	apply(&opts)
	s.cfg.Profile = mustProfile(opts)
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

func (s *generatorScenario) withPlanOnly() *generatorScenario {
	s.t.Helper()
	s.applyEnabled = false
	return s
}

func (s *generatorScenario) generator() *Generator {
	s.t.Helper()
	depGraph := citest.DependencyGraph(s.modules, s.dependencies)
	return newTestGeneratorWithTargetsAndApply(s.t, s.cfg.GitLab, s.cfg.Profile, s.cfg.Contributions, depGraph, s.modules, s.generateTargets(), s.applyEnabled)
}

func (s *generatorScenario) generate() *Pipeline {
	s.t.Helper()
	genPipeline, err := s.generator().Generate()
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
	result, err := s.generator().DryRun()
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

func mustJob(t *testing.T, gitlabPipeline *Pipeline, name string) Job {
	t.Helper()
	job, ok := gitlabPipeline.Job(name)
	if !ok {
		t.Fatalf("job %q not found", name)
	}
	return job
}
