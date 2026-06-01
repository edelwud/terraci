package generate

import (
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
)

// newTestGenerator builds a canonical IR and constructs the Generator. Tests
// use this so they don't have to thread the IR through scenario builders.
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

func newTestGeneratorWithTargetsAndApply(
	tb testing.TB,
	cfg *configpkg.Config,
	execCfg execution.Config,
	contributions []*pipeline.Contribution,
	depGraph *graph.DependencyGraph,
	allModules, targetModules []*discovery.Module,
	applyEnabled bool,
) *Generator {
	tb.Helper()
	ir := mustBuildIRWithApply(tb, cfg, execCfg, contributions, depGraph, allModules, targetModules, applyEnabled)
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
	ir, err := buildTestIR(cfg, execCfg, contributions, depGraph, allModules, targetModules)
	if err != nil {
		tb.Fatalf("buildTestIR() error = %v", err)
	}
	return ir
}

func mustBuildIRWithApply(
	tb testing.TB,
	cfg *configpkg.Config,
	execCfg execution.Config,
	contributions []*pipeline.Contribution,
	depGraph *graph.DependencyGraph,
	allModules, targetModules []*discovery.Module,
	applyEnabled bool,
) *pipeline.IR {
	tb.Helper()
	ir, err := buildTestIRWithApply(cfg, execCfg, contributions, depGraph, allModules, targetModules, applyEnabled)
	if err != nil {
		tb.Fatalf("buildTestIRWithApply() error = %v", err)
	}
	return ir
}
