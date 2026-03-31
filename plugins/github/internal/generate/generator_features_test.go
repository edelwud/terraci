package generate

import (
	"testing"

	"github.com/edelwud/terraci/pkg/pipeline"
	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
)

func TestGenerate_WithPR(t *testing.T) {
	vpc := createTestModule("platform", "stage", "eu-central-1", "vpc")
	eks := createTestModule("platform", "stage", "eu-central-1", "eks")
	workflow := newGeneratorScenario(t).
		withConfig(func(cfg *configpkg.Config) { cfg.PR = &configpkg.PRConfig{} }).
		withContributions([]*pipeline.Contribution{{
			Jobs: []pipeline.ContributedJob{{
				Name:          "terraci-summary",
				Phase:         pipeline.PhaseFinalize,
				Commands:      []string{"terraci summary"},
				DependsOnPlan: true,
				AllowFailure:  false,
			}},
		}}).
		withModules(vpc, eks).
		withDependencies(map[string][]string{
			vpc.ID(): {},
			eks.ID(): {vpc.ID()},
		}).
		generate()

	assertWorkflow(t, workflow).hasJob("terraci-summary")
	assertWorkflow(t, workflow).
		job("terraci-summary").
		hasNeed("plan-platform-stage-eu-central-1-vpc").
		hasNeed("plan-platform-stage-eu-central-1-eks")
}

func TestGenerate_WithPolicy(t *testing.T) {
	module := createTestModule("platform", "stage", "eu-central-1", "vpc")
	workflow := newGeneratorScenario(t).
		withContributions([]*pipeline.Contribution{{
			Jobs: []pipeline.ContributedJob{{
				Name:          "policy-check",
				Phase:         pipeline.PhasePostPlan,
				Commands:      []string{"terraci policy pull", "terraci policy check"},
				ArtifactPaths: []string{".terraci/policy-results.json"},
				DependsOnPlan: true,
				AllowFailure:  false,
			}},
		}}).
		withModules(module).
		withDependencies(map[string][]string{module.ID(): {}}).
		generate()

	assertWorkflow(t, workflow).
		hasJob("policy-check").
		job("policy-check").
		hasNeed("plan-platform-stage-eu-central-1-vpc")
}
