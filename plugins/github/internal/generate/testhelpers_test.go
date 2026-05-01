package generate

import (
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
)

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
