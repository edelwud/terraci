// Package workflow provides shared orchestration logic for module discovery,
// filtering, dependency extraction, and graph building.
package workflow

import (
	"context"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/filter"
	"github.com/edelwud/terraci/internal/graph"
	"github.com/edelwud/terraci/internal/parser"
	terrierrors "github.com/edelwud/terraci/pkg/errors"
	"github.com/edelwud/terraci/pkg/log"
)

// Options configures module discovery, filtering, and graph building.
type Options struct {
	WorkDir  string
	MinDepth int
	MaxDepth int
	Segments []string

	Excludes       []string
	Includes       []string
	SegmentFilters map[string][]string
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
func Run(ctx context.Context, opts Options) (*Result, error) {
	scanner := discovery.NewScanner(opts.WorkDir, opts.MinDepth, opts.MaxDepth, opts.Segments)

	allModules, err := scanner.Scan(ctx)
	if err != nil {
		return nil, &terrierrors.ScanError{Dir: opts.WorkDir, Err: err}
	}

	log.WithField("count", len(allModules)).Info("discovered modules")

	if len(allModules) == 0 {
		return nil, &terrierrors.NoModulesError{Dir: opts.WorkDir}
	}

	filtered := filter.Apply(allModules, filter.Options{
		Excludes: opts.Excludes,
		Includes: opts.Includes,
		Segments: opts.SegmentFilters,
	})

	if len(filtered) != len(allModules) {
		log.WithField("before", len(allModules)).WithField("after", len(filtered)).Info("filtered modules")
	}

	fullIndex := discovery.NewModuleIndex(allModules)
	filteredIndex := discovery.NewModuleIndex(filtered)

	hclParser := parser.NewParser(opts.Segments)

	deps, warnings := parser.NewDependencyExtractor(hclParser, filteredIndex).ExtractAllDependencies(ctx)
	depGraph := graph.BuildFromDependencies(filtered, deps)

	return &Result{
		AllModules:      allModules,
		FilteredModules: filtered,
		FullIndex:       fullIndex,
		FilteredIndex:   filteredIndex,
		Graph:           depGraph,
		Dependencies:    deps,
		Warnings:        warnings,
	}, nil
}
