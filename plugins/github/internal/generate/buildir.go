package generate

import (
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
)

// BuildPipelineIR builds the canonical pipeline IR for the GitHub generator
// from raw inputs. Production callers normally build the IR via core
// (cmd/terraci) and pass it to NewGenerator directly; this helper lets tests
// construct an IR with the same script-config / plan-only / detailed-plan
// semantics that the GitHub plugin would derive from its own settings.
func BuildPipelineIR(
	cfg *configpkg.Config,
	execCfg execution.Config,
	contributions []*pipeline.Contribution,
	depGraph *graph.DependencyGraph,
	allModules, targetModules []*discovery.Module,
) (*pipeline.IR, error) {
	s := newSettings(cfg, execCfg)
	return pipeline.Build(pipeline.BuildOptions{
		DepGraph:      depGraph,
		TargetModules: targetModules,
		AllModules:    allModules,
		ModuleIndex:   discovery.NewModuleIndex(allModules),
		Script: pipeline.ScriptConfig{
			InitEnabled:  s.initEnabled(),
			PlanEnabled:  s.planEnabled(),
			AutoApprove:  s.autoApprove(),
			DetailedPlan: s.prEnabled(),
		},
		Contributions: contributions,
		PlanEnabled:   s.planEnabled(),
		PlanOnly:      s.planOnly(),
	})
}
