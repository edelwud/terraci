// Package pipelinetest provides shared helpers for IR-construction in plugin
// tests. Production callers build the IR via cmd/terraci's flow and pass the
// finished *pipeline.IR to provider generators; tests use these helpers to
// reproduce that shape without reimplementing the BuildOptions plumbing.
package pipelinetest

import (
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
)

// IROptions describes the test inputs for BuildIR. The fields mirror the
// subset of pipeline.BuildOptions that test scenarios actually drive — the
// rest (ModuleIndex, etc.) is derived automatically.
type IROptions struct {
	// Script is the rendering knob set provider plugins compute from their
	// own settings (e.g. DetailedPlan from "MR comment enabled?"). Tests
	// construct it directly so the helper is provider-agnostic.
	Script pipeline.ScriptConfig

	// Contributions are merged into the IR, same as production.
	Contributions []*pipeline.Contribution

	// DepGraph and the module slices come from the test's discovery setup.
	// AllModules covers the universe; TargetModules is the filtered run set.
	DepGraph      *graph.DependencyGraph
	AllModules    []*discovery.Module
	TargetModules []*discovery.Module

	// PlanEnabled / PlanOnly mirror the BuildOptions fields one-to-one.
	PlanEnabled bool
	PlanOnly    bool
}

// BuildIR constructs a pipeline.IR for tests using the supplied IROptions.
// Replaces near-identical BuildPipelineIR helpers previously duplicated in
// each provider plugin's internal/generate package.
func BuildIR(opts IROptions) (*pipeline.IR, error) {
	return pipeline.Build(pipeline.BuildOptions{
		DepGraph:      opts.DepGraph,
		TargetModules: opts.TargetModules,
		AllModules:    opts.AllModules,
		ModuleIndex:   discovery.NewModuleIndex(opts.AllModules),
		Script:        opts.Script,
		Contributions: opts.Contributions,
		PlanEnabled:   opts.PlanEnabled,
		PlanOnly:      opts.PlanOnly,
	})
}
