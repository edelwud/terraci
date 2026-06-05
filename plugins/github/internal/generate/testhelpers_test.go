package generate

import (
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
)

func defaultTerraformConfigOptions() pipeline.TerraformJobConfigOptions {
	return pipeline.TerraformJobConfigOptions{
		Binary:      "terraform",
		InitEnabled: true,
	}
}

func newTestGeneratorWithTargetsAndApply(
	tb testing.TB,
	cfg *configpkg.Config,
	terraformConfig pipeline.TerraformJobConfigOptions,
	contributions pipeline.ContributionSet,
	depGraph *graph.DependencyGraph,
	allModules, targetModules []*discovery.Module,
	applyEnabled bool,
) *Generator {
	tb.Helper()
	ir := mustBuildIRWithApply(tb, cfg, terraformConfig, contributions, depGraph, allModules, targetModules, applyEnabled)
	return NewGenerator(cfg, ir)
}

func mustBuildIRWithApply(
	tb testing.TB,
	cfg *configpkg.Config,
	terraformConfig pipeline.TerraformJobConfigOptions,
	contributions pipeline.ContributionSet,
	depGraph *graph.DependencyGraph,
	allModules, targetModules []*discovery.Module,
	applyEnabled bool,
) *pipeline.IR {
	tb.Helper()
	ir, err := buildTestIRWithApply(cfg, terraformConfig, contributions, depGraph, allModules, targetModules, applyEnabled)
	if err != nil {
		tb.Fatalf("buildTestIRWithApply() error = %v", err)
	}
	return ir
}
