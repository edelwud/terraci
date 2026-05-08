package generate

import (
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
)

func buildTestIR(
	cfg *configpkg.Config,
	execCfg execution.Config,
	contributions []*pipeline.Contribution,
	depGraph *graph.DependencyGraph,
	allModules, targetModules []*discovery.Module,
) (*pipeline.IR, error) {
	s := newSettings(cfg, execCfg)
	requirements := execCfg.BuildRequirements().Merge(PipelineRequirements(cfg))
	return pipeline.Build(pipeline.BuildOptions{
		DepGraph:      depGraph,
		TargetModules: targetModules,
		AllModules:    allModules,
		ModuleIndex:   discovery.NewModuleIndex(allModules),
		Script: pipeline.ScriptConfig{
			InitEnabled: s.initEnabled(),
			PlanEnabled: s.planEnabled(),
		},
		Contributions: contributions,
		Requirements:  requirements,
		PlanEnabled:   s.planEnabled(),
	})
}
