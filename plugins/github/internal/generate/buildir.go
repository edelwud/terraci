package generate

import (
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/pipeline/pipelinetest"
	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
)

// BuildPipelineIR builds the canonical pipeline IR for the GitHub generator
// from raw inputs. Runtime requirements come from execution config and
// provider-specific requirements come from GitHub settings; the actual IR
// construction goes through pipelinetest.BuildIR so it stays aligned with the
// GitLab equivalent.
func BuildPipelineIR(
	cfg *configpkg.Config,
	execCfg execution.Config,
	contributions []*pipeline.Contribution,
	depGraph *graph.DependencyGraph,
	allModules, targetModules []*discovery.Module,
) (*pipeline.IR, error) {
	s := newSettings(cfg, execCfg)
	requirements := execCfg.BuildRequirements().Merge(PipelineRequirements(cfg))
	return pipelinetest.BuildIR(pipelinetest.IROptions{
		Script: pipeline.ScriptConfig{
			InitEnabled: s.initEnabled(),
			PlanEnabled: s.planEnabled(),
		},
		Contributions: contributions,
		Requirements:  requirements,
		DepGraph:      depGraph,
		AllModules:    allModules,
		TargetModules: targetModules,
		PlanEnabled:   s.planEnabled(),
	})
}

func PipelineRequirements(cfg *configpkg.Config) pipeline.BuildRequirements {
	return pipeline.BuildRequirements{PlanOnly: newSettings(cfg, execution.Config{}).planOnly()}
}
