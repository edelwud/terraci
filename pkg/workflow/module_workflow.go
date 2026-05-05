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

	// LibraryPaths are project-relative roots whose discovered modules will be
	// flagged Module.IsLibrary=true and routed into Result.Libraries instead
	// of the executable target sets. Empty/nil disables the feature.
	LibraryPaths []string
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
	// All contains every discovered module (executable + library), in scan
	// order. Use Filtered for execution targets and Libraries for reporting.
	All ModuleSet
	// Filtered is the executable subset after filters; library modules are
	// always excluded regardless of filters.
	Filtered ModuleSet
	// Libraries holds modules under any configured library_modules.paths.
	// They are not included in Filtered and never become execution targets,
	// but are tracked here for diagnostics (validate/graph).
	Libraries ModuleSet

	Graph        *graph.DependencyGraph
	Dependencies map[string]*parser.ModuleDependencies
	Warnings     []error
}

// Run executes the full module workflow: scan → filter → parse → build graph.
func Run(ctx context.Context, opts Options) (*Result, error) {
	scanner := discovery.NewScanner(opts.WorkDir, opts.Segments, opts.LibraryPaths...)

	allModules, err := scanner.Scan(ctx)
	if err != nil {
		return nil, &terrierrors.ScanError{Dir: opts.WorkDir, Err: err}
	}

	log.WithField("count", len(allModules)).Info("discovered modules")

	if len(allModules) == 0 {
		return nil, &terrierrors.NoModulesError{Dir: opts.WorkDir}
	}

	executable, libraries := splitLibraries(allModules)

	filtered := filter.Apply(executable, filter.Options{
		Excludes: opts.Excludes,
		Includes: opts.Includes,
		Segments: opts.SegmentFilters,
	})

	if len(filtered) != len(executable) {
		log.WithField("before", len(executable)).WithField("after", len(filtered)).Info("filtered modules")
	}
	if len(libraries) > 0 {
		log.WithField("count", len(libraries)).Debug("identified library modules")
	}

	allSet := NewModuleSet(allModules)
	filteredSet := NewModuleSet(filtered)
	librarySet := NewModuleSet(libraries)

	hclParser := parser.NewParser(opts.Segments)

	deps, warnings := parser.NewDependencyExtractor(hclParser, filteredSet.Index).ExtractAllDependencies(ctx)

	depGraph := graph.BuildFromDependencies(filtered, deps)

	return &Result{
		All:          allSet,
		Filtered:     filteredSet,
		Libraries:    librarySet,
		Graph:        depGraph,
		Dependencies: deps,
		Warnings:     warnings,
	}, nil
}

// splitLibraries partitions discovered modules into executable and library
// subsets while preserving original order in each. Library modules are those
// with Module.IsLibrary=true, set by the scanner from cleaned LibraryPaths.
func splitLibraries(modules []*discovery.Module) (executable, libraries []*discovery.Module) {
	for _, m := range modules {
		if m.IsLibrary {
			libraries = append(libraries, m)
			continue
		}
		executable = append(executable, m)
	}
	return executable, libraries
}
