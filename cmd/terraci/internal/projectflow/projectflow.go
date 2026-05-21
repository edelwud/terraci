// Package projectflow owns command-side Terraform project discovery.
package projectflow

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/edelwud/terraci/cmd/terraci/internal/runflow"
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/workflow"
)

var errPreparedRequired = errors.New("projectflow prepared state is required")

// Runtime contains immutable command state needed to discover a Terraform project.
type Runtime struct {
	prepared *runflow.Prepared
}

// NewRuntime creates a project discovery runtime from prepared command state.
func NewRuntime(prepared *runflow.Prepared) Runtime {
	return Runtime{prepared: prepared}
}

// Request describes one project discovery request.
type Request struct {
	Filters       filter.Flags
	SelectTargets bool
	ChangedOnly   bool
	BaseRef       string
}

// Result contains project discovery output plus command-facing diagnostics.
type Result struct {
	Workflow       *workflow.Result
	Targets        []*discovery.Module
	LibraryUsages  []LibraryUsage
	LibrarySummary *LibrarySummary
}

// LibraryUsage describes one tracked library module path and its consumers.
type LibraryUsage struct {
	Path         string
	RelativePath string
	UsedBy       int
}

// LibrarySummary describes discovered library_modules diagnostics.
type LibrarySummary struct {
	ConfiguredPaths int
	Discovered      int
	Consumers       int
	Orphans         []string
}

// Run scans, filters, parses, and optionally resolves executable targets.
func Run(ctx context.Context, runtime Runtime, req Request) (*Result, error) {
	if runtime.prepared == nil {
		return nil, errPreparedRequired
	}
	flags := cloneFilterFlags(req.Filters)
	result, err := workflow.Run(ctx, workflow.OptionsFromConfig(runtime.prepared.WorkDir(), runtime.prepared.Config(), flags))
	if err != nil {
		return nil, err
	}

	out := &Result{
		Workflow:       result,
		LibraryUsages:  LibraryUsages(runtime, result),
		LibrarySummary: SummarizeLibraries(runtime.prepared.Config(), result),
	}
	if req.SelectTargets {
		targets, err := ResolveTargets(ctx, runtime, result, req)
		if err != nil {
			return nil, err
		}
		out.Targets = targets
	}
	return out, nil
}

// ResolveTargets applies command target-selection semantics to a workflow result.
func ResolveTargets(ctx context.Context, runtime Runtime, result *workflow.Result, req Request) ([]*discovery.Module, error) {
	if runtime.prepared == nil {
		return nil, errPreparedRequired
	}
	appCtx := runtime.prepared.AppContext()
	flags := cloneFilterFlags(req.Filters)
	return workflow.ResolveTargets(ctx, runtime.prepared.WorkDir(), runtime.prepared.Config(), result, workflow.TargetSelectionOptions{
		ChangedOnly: req.ChangedOnly,
		BaseRef:     req.BaseRef,
		Filters:     flags,
		ChangeDetectorResolver: func() (workflow.ChangeDetector, error) {
			return appCtx.ChangeDetectorResolver().ResolveChangeDetector()
		},
	})
}

// LibraryUsages derives deterministic library usage diagnostics.
func LibraryUsages(runtime Runtime, result *workflow.Result) []LibraryUsage {
	if runtime.prepared == nil || result == nil || result.Graph == nil {
		return nil
	}
	paths := result.Graph.GetAllLibraryPaths()
	if len(paths) == 0 {
		return nil
	}
	usages := make([]LibraryUsage, 0, len(paths))
	for _, path := range paths {
		usages = append(usages, LibraryUsage{
			Path:         path,
			RelativePath: makeRelative(path, runtime.prepared.WorkDir()),
			UsedBy:       len(result.Graph.GetModulesUsingLibrary(path)),
		})
	}
	return usages
}

// SummarizeLibraries derives configured library_modules diagnostics.
func SummarizeLibraries(cfg config.Snapshot, result *workflow.Result) *LibrarySummary {
	libraryModules := cfg.LibraryModules()
	if !cfg.Present() || libraryModules == nil || len(libraryModules.Paths) == 0 || result == nil || result.Graph == nil {
		return nil
	}
	orphans := make([]string, 0)
	for _, module := range result.Libraries.Modules {
		if !result.Graph.HasLibraryConsumers(module.Path) {
			orphans = append(orphans, module.RelativePath)
		}
	}
	return &LibrarySummary{
		ConfiguredPaths: len(libraryModules.Paths),
		Discovered:      len(result.Libraries.Modules),
		Consumers:       result.Graph.LibraryConsumerCount(),
		Orphans:         orphans,
	}
}

func cloneFilterFlags(flags filter.Flags) *filter.Flags {
	return &filter.Flags{
		Excludes:    append([]string(nil), flags.Excludes...),
		Includes:    append([]string(nil), flags.Includes...),
		SegmentArgs: append([]string(nil), flags.SegmentArgs...),
	}
}

func makeRelative(path, base string) string {
	if absBase, err := filepath.Abs(base); err == nil {
		if rel, err := filepath.Rel(absBase, path); err == nil {
			return rel
		}
	}
	return path
}
