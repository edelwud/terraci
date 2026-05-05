// Package workflow provides shared orchestration logic for module discovery,
// filtering, dependency extraction, and graph building.
package workflow

import (
	"context"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/discovery"
	terrierrors "github.com/edelwud/terraci/pkg/errors"
	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/parser"
)

// Options configures module discovery, filtering, and graph building.
type Options struct {
	WorkDir  string
	Segments []string

	Excludes       []string
	Includes       []string
	SegmentFilters map[string][]string
}

// ModuleSet keeps a module slice and its lookup index together.
type ModuleSet struct {
	Modules []*discovery.Module
	Index   *discovery.ModuleIndex
}

// NewModuleSet builds a consistent module collection and lookup index.
func NewModuleSet(modules []*discovery.Module) ModuleSet {
	return ModuleSet{
		Modules: modules,
		Index:   discovery.NewModuleIndex(modules),
	}
}

// All returns modules in their workflow order.
func (s ModuleSet) All() []*discovery.Module {
	if len(s.Modules) == 0 && s.Index != nil {
		return s.Index.All()
	}
	return s.Modules
}

// ByID returns a module by workflow ID.
func (s ModuleSet) ByID(id string) *discovery.Module {
	if s.Index == nil {
		return nil
	}
	return s.Index.ByID(id)
}

// Result contains everything produced by the module workflow.
type Result struct {
	All      ModuleSet
	Filtered ModuleSet

	Graph        *graph.DependencyGraph
	Dependencies map[string]*parser.ModuleDependencies
	Warnings     []error
}

// Run executes the full module workflow: scan → filter → parse → build graph.
func Run(ctx context.Context, opts Options) (*Result, error) {
	scanner := discovery.NewScanner(opts.WorkDir, opts.Segments)

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

	allSet := NewModuleSet(allModules)
	filteredSet := NewModuleSet(filtered)

	hclParser := parser.NewParser(opts.Segments)

	deps, warnings := parser.NewDependencyExtractor(hclParser, filteredSet.Index).ExtractAllDependencies(ctx)

	depGraph := graph.BuildFromDependencies(filtered, deps)

	return &Result{
		All:          allSet,
		Filtered:     filteredSet,
		Graph:        depGraph,
		Dependencies: deps,
		Warnings:     warnings,
	}, nil
}
