package generate

import (
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/terraformrun"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
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

// newTestGenerator builds a canonical IR and constructs the Generator. Tests
// use this so they don't have to thread the IR through scenario builders.
func newTestGenerator(
	tb testing.TB,
	cfg *configpkg.Config,
	profile terraformrun.Profile,
	contributions []*pipeline.Contribution,
	depGraph *graph.DependencyGraph,
	allModules []*discovery.Module,
) *Generator {
	tb.Helper()
	ir := mustBuildIR(tb, cfg, profile, contributions, depGraph, allModules, nil)
	return NewGenerator(cfg, profile, ir)
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

func mustBuildIR(
	tb testing.TB,
	cfg *configpkg.Config,
	profile terraformrun.Profile,
	contributions []*pipeline.Contribution,
	depGraph *graph.DependencyGraph,
	allModules, targetModules []*discovery.Module,
) *pipeline.IR {
	tb.Helper()
	ir, err := buildTestIR(cfg, profile, contributions, depGraph, allModules, targetModules)
	if err != nil {
		tb.Fatalf("buildTestIR() error = %v", err)
	}
	return ir
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
