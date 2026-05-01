package generate

import (
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
)

// newTestGenerator builds the IR via the public BuildPipelineIR helper and
// constructs the Generator. Tests use this so they don't have to thread the
// IR through scenario builders.
func newTestGenerator(
	tb testing.TB,
	cfg *configpkg.Config,
	execCfg execution.Config,
	contributions []*pipeline.Contribution,
	depGraph *graph.DependencyGraph,
	allModules []*discovery.Module,
) *Generator {
	tb.Helper()
	ir := mustBuildIR(tb, cfg, execCfg, contributions, depGraph, allModules, nil)
	return NewGenerator(cfg, execCfg, ir)
}

func newTestGeneratorWithTargets(
	tb testing.TB,
	cfg *configpkg.Config,
	execCfg execution.Config,
	contributions []*pipeline.Contribution,
	depGraph *graph.DependencyGraph,
	allModules, targetModules []*discovery.Module,
) *Generator {
	tb.Helper()
	ir := mustBuildIR(tb, cfg, execCfg, contributions, depGraph, allModules, targetModules)
	return NewGenerator(cfg, execCfg, ir)
}

func mustBuildIR(
	tb testing.TB,
	cfg *configpkg.Config,
	execCfg execution.Config,
	contributions []*pipeline.Contribution,
	depGraph *graph.DependencyGraph,
	allModules, targetModules []*discovery.Module,
) *pipeline.IR {
	tb.Helper()
	ir, err := BuildPipelineIR(cfg, execCfg, contributions, depGraph, allModules, targetModules)
	if err != nil {
		tb.Fatalf("BuildPipelineIR() error = %v", err)
	}
	return ir
}
