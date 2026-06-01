package generate

import (
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/terraformrun"
	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
)

func mustProfile(opts terraformrun.ProfileOptions) terraformrun.Profile {
	profile, err := terraformrun.NewProfile(opts)
	if err != nil {
		panic(err)
	}
	return profile
}

func profileOptionsFromProfile(profile terraformrun.Profile) terraformrun.ProfileOptions {
	initEnabled := profile.InitEnabled()
	return terraformrun.ProfileOptions{
		Binary:      profile.Binary().String(),
		InitEnabled: &initEnabled,
		Parallelism: profile.Parallelism(),
		Env:         profile.Env(),
	}
}

func newTestGeneratorWithTargetsAndApply(
	tb testing.TB,
	cfg *configpkg.Config,
	profile terraformrun.Profile,
	contributions []*pipeline.Contribution,
	depGraph *graph.DependencyGraph,
	allModules, targetModules []*discovery.Module,
	applyEnabled bool,
) *Generator {
	tb.Helper()
	ir := mustBuildIRWithApply(tb, cfg, profile, contributions, depGraph, allModules, targetModules, applyEnabled)
	return NewGenerator(cfg, profile, ir)
}

func mustBuildIRWithApply(
	tb testing.TB,
	cfg *configpkg.Config,
	profile terraformrun.Profile,
	contributions []*pipeline.Contribution,
	depGraph *graph.DependencyGraph,
	allModules, targetModules []*discovery.Module,
	applyEnabled bool,
) *pipeline.IR {
	tb.Helper()
	ir, err := buildTestIRWithApply(cfg, profile, contributions, depGraph, allModules, targetModules, applyEnabled)
	if err != nil {
		tb.Fatalf("buildTestIRWithApply() error = %v", err)
	}
	return ir
}
