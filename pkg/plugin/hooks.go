package plugin

import (
	"context"
	"sort"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/parser"
)

// HookPhase identifies where in the workflow a hook runs.
type HookPhase int

const (
	// PhasePreScan runs before module discovery.
	PhasePreScan HookPhase = iota
	// PhasePostScan runs after module discovery, before filtering.
	PhasePostScan
	// PhasePostFilter runs after filtering, before HCL parsing.
	PhasePostFilter
	// PhasePostParse runs after HCL parsing, before graph building.
	PhasePostParse
	// PhasePostGraph runs after the dependency graph is built.
	PhasePostGraph
)

// WorkflowHook injects behavior at a specific workflow phase.
type WorkflowHook struct {
	Phase    HookPhase
	Priority int // Lower = runs first. Default 100.
	Fn       func(ctx context.Context, state *WorkflowState) error
}

// WorkflowState provides mutable access to intermediate workflow results.
// Fields are populated progressively as the workflow advances through phases.
type WorkflowState struct {
	// AllModules is populated after scan.
	AllModules []*discovery.Module
	// Filtered is populated after filter.
	Filtered []*discovery.Module
	// Dependencies is populated after parse.
	Dependencies map[string]*parser.ModuleDependencies
	// Graph is populated after graph build.
	Graph *graph.DependencyGraph
	// Warnings collects non-fatal issues.
	Warnings []error
}

// CollectHooks gathers all workflow hooks from registered plugins, sorted by priority.
func CollectHooks() []WorkflowHook {
	var hooks []WorkflowHook
	for _, p := range ByCapability[WorkflowHookProvider]() {
		hooks = append(hooks, p.WorkflowHooks()...)
	}
	sort.Slice(hooks, func(i, j int) bool {
		return hooks[i].Priority < hooks[j].Priority
	})
	return hooks
}

// RunHooks executes all hooks for the given phase.
func RunHooks(ctx context.Context, hooks []WorkflowHook, phase HookPhase, state *WorkflowState) error {
	for _, h := range hooks {
		if h.Phase == phase {
			if err := h.Fn(ctx, state); err != nil {
				return err
			}
		}
	}
	return nil
}
