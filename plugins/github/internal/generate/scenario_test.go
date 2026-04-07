package generate

import (
	"testing"

	"github.com/edelwud/terraci/pkg/ci/citest"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/pipeline"
	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
	domainpkg "github.com/edelwud/terraci/plugins/github/internal/domain"
)

func createTestModule(service, env, region, module string) *discovery.Module {
	return citest.TestModule(service, env, region, module)
}

type testCfg struct {
	GitHub        *configpkg.Config
	Contributions []*pipeline.Contribution
}

func createTestConfig() *testCfg {
	return &testCfg{
		GitHub: &configpkg.Config{
			RunsOn:      "ubuntu-latest",
			PlanEnabled: true,
			InitEnabled: true,
		},
	}
}

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

func (s *generatorScenario) withConfig(apply func(*configpkg.Config)) *generatorScenario {
	s.t.Helper()
	apply(s.cfg.GitHub)
	return s
}

func (s *generatorScenario) withContributions(contributions []*pipeline.Contribution) *generatorScenario {
	s.t.Helper()
	s.cfg.Contributions = contributions
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

func (s *generatorScenario) generator() *Generator {
	s.t.Helper()
	depGraph := citest.DependencyGraph(s.modules, s.dependencies)
	return NewGenerator(s.cfg.GitHub, s.cfg.Contributions, depGraph, s.modules)
}

func (s *generatorScenario) generate() *domainpkg.Workflow {
	s.t.Helper()
	result, err := s.generator().Generate(s.generateTargets())
	if err != nil {
		s.t.Fatalf("Generate failed: %v", err)
	}
	workflow, ok := result.(*domainpkg.Workflow)
	if !ok {
		s.t.Fatal("expected *Workflow type")
	}
	return workflow
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
