package generate

import (
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
)

type fixtureScenario struct {
	t       *testing.T
	fixture *Fixture
	targets []*discovery.Module
}

func newFixtureScenario(t *testing.T, name string) *fixtureScenario {
	t.Helper()
	return &fixtureScenario{
		t:       t,
		fixture: LoadFixture(t, name),
	}
}

func (s *fixtureScenario) withConfig(apply func(*Config)) *fixtureScenario {
	s.t.Helper()
	apply(s.fixture.GLConfig)
	s.fixture.Generator = NewGenerator(s.fixture.GLConfig, s.fixture.ExecConfig, s.fixture.Contributions, s.fixture.DepGraph, s.fixture.Modules)
	return s
}

func (s *fixtureScenario) withExecution(apply func(*execution.Config)) *fixtureScenario {
	s.t.Helper()
	apply(&s.fixture.ExecConfig)
	s.fixture.Generator = NewGenerator(s.fixture.GLConfig, s.fixture.ExecConfig, s.fixture.Contributions, s.fixture.DepGraph, s.fixture.Modules)
	return s
}

func (s *fixtureScenario) withEnvironment(environment string) *fixtureScenario {
	s.t.Helper()
	s.targets = s.fixture.GetModulesByEnvironment(environment)
	return s
}

func (s *fixtureScenario) withTargets(targets ...*discovery.Module) *fixtureScenario {
	s.t.Helper()
	s.targets = targets
	return s
}

func (s *fixtureScenario) withTargetNames(names ...string) *fixtureScenario {
	s.t.Helper()
	targets := make([]*discovery.Module, 0, len(names))
	for _, name := range names {
		module := s.fixture.GetModuleByName(name)
		if module == nil {
			s.t.Fatalf("module %q not found in fixture %q", name, s.fixture.Name)
		}
		targets = append(targets, module)
	}
	s.targets = targets
	return s
}

func (s *fixtureScenario) generate() *Pipeline {
	s.t.Helper()
	targets := s.targets
	if targets == nil {
		targets = s.fixture.Modules
	}

	result, err := s.fixture.Generator.Generate(targets)
	if err != nil {
		s.t.Fatalf("failed to generate pipeline: %v", err)
	}

	pipeline, ok := result.(*Pipeline)
	if !ok {
		s.t.Fatal("expected *Pipeline type")
	}
	return pipeline
}
