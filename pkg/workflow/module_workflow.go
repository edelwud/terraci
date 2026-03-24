// Package workflow provides shared orchestration logic for module discovery,
// filtering, dependency extraction, and graph building.
package workflow

import (
	"context"

	"github.com/edelwud/terraci/pkg/discovery"
	terrierrors "github.com/edelwud/terraci/pkg/errors"
	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/parser"
	"github.com/edelwud/terraci/pkg/plugin"
)

// Options configures module discovery, filtering, and graph building.
type Options struct {
	WorkDir  string
	Segments []string

	Excludes       []string
	Includes       []string
	SegmentFilters map[string][]string

	// Hooks are optional workflow hooks executed at each phase.
	// If nil, hooks are collected from registered plugins automatically.
	Hooks []plugin.WorkflowHook
}

// Result contains everything produced by the module workflow.
type Result struct {
	AllModules      []*discovery.Module
	FilteredModules []*discovery.Module
	FullIndex       *discovery.ModuleIndex
	FilteredIndex   *discovery.ModuleIndex
	Graph           *graph.DependencyGraph
	Dependencies    map[string]*parser.ModuleDependencies
	Warnings        []error
}

// Run executes the full module workflow: scan → filter → parse → build graph.
// Workflow hooks are executed at each phase if provided in opts or registered via plugins.
func Run(ctx context.Context, opts Options) (*Result, error) {
	hooks := opts.Hooks
	if hooks == nil {
		hooks = plugin.CollectHooks()
	}

	state := &plugin.WorkflowState{}

	// Pre-scan hooks
	if err := plugin.RunHooks(ctx, hooks, plugin.PhasePreScan, state); err != nil {
		return nil, err
	}

	scanner := discovery.NewScanner(opts.WorkDir, opts.Segments)

	allModules, err := scanner.Scan(ctx)
	if err != nil {
		return nil, &terrierrors.ScanError{Dir: opts.WorkDir, Err: err}
	}

	log.WithField("count", len(allModules)).Info("discovered modules")

	if len(allModules) == 0 {
		return nil, &terrierrors.NoModulesError{Dir: opts.WorkDir}
	}

	state.AllModules = allModules

	// Post-scan hooks
	if err := plugin.RunHooks(ctx, hooks, plugin.PhasePostScan, state); err != nil {
		return nil, err
	}

	filtered := filter.Apply(allModules, filter.Options{
		Excludes: opts.Excludes,
		Includes: opts.Includes,
		Segments: opts.SegmentFilters,
	})

	if len(filtered) != len(allModules) {
		log.WithField("before", len(allModules)).WithField("after", len(filtered)).Info("filtered modules")
	}

	state.Filtered = filtered

	// Post-filter hooks
	if err := plugin.RunHooks(ctx, hooks, plugin.PhasePostFilter, state); err != nil {
		return nil, err
	}

	fullIndex := discovery.NewModuleIndex(allModules)
	filteredIndex := discovery.NewModuleIndex(filtered)

	hclParser := parser.NewParser(opts.Segments)

	deps, warnings := parser.NewDependencyExtractor(hclParser, filteredIndex).ExtractAllDependencies(ctx)

	state.Dependencies = deps
	state.Warnings = warnings

	// Post-parse hooks
	if err := plugin.RunHooks(ctx, hooks, plugin.PhasePostParse, state); err != nil {
		return nil, err
	}

	depGraph := graph.BuildFromDependencies(filtered, deps)

	state.Graph = depGraph

	// Post-graph hooks
	if err := plugin.RunHooks(ctx, hooks, plugin.PhasePostGraph, state); err != nil {
		return nil, err
	}

	return &Result{
		AllModules:      allModules,
		FilteredModules: state.Filtered,
		FullIndex:       fullIndex,
		FilteredIndex:   filteredIndex,
		Graph:           state.Graph,
		Dependencies:    state.Dependencies,
		Warnings:        state.Warnings,
	}, nil
}
