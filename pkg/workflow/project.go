package workflow

import (
	"context"
	"path/filepath"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/filter"
)

// ProjectRequest describes one canonical Terraform project planning request.
type ProjectRequest struct {
	WorkDir   string
	Config    config.Config
	Filters   filter.Flags
	Targeting TargetRequest
}

// TargetRequest controls optional executable target selection.
type TargetRequest struct {
	Enabled bool

	ModulePath  string
	ChangedOnly bool
	BaseRef     string

	ChangeDetectorResolver ChangeDetectorResolver
}

// ProjectResult contains workflow output plus target selection and diagnostics.
type ProjectResult struct {
	Workflow       *Result
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

// PlanProject runs workflow discovery and optional target selection from one
// canonical request.
func PlanProject(ctx context.Context, req ProjectRequest) (*ProjectResult, error) {
	cfg := req.Config
	if !cfg.Present() {
		cfg = config.Default()
	}
	flags := cloneFilterFlags(req.Filters)

	result, err := run(ctx, optionsFromConfig(req.WorkDir, cfg, flags))
	if err != nil {
		return nil, err
	}

	out := &ProjectResult{
		Workflow:       result,
		LibraryUsages:  LibraryUsages(req.WorkDir, result),
		LibrarySummary: SummarizeLibraries(cfg, result),
	}
	if req.Targeting.Enabled {
		targets, err := resolveTargets(ctx, req.WorkDir, cfg, result, targetSelectionOptions{
			ModulePath:             req.Targeting.ModulePath,
			ChangedOnly:            req.Targeting.ChangedOnly,
			BaseRef:                req.Targeting.BaseRef,
			Filters:                flags,
			ChangeDetectorResolver: req.Targeting.ChangeDetectorResolver,
		})
		if err != nil {
			return nil, err
		}
		out.Targets = targets
	}
	return out, nil
}

// LibraryUsages derives deterministic library usage diagnostics.
func LibraryUsages(workDir string, result *Result) []LibraryUsage {
	if result == nil || result.Graph == nil {
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
			RelativePath: makeRelative(path, workDir),
			UsedBy:       len(result.Graph.GetModulesUsingLibrary(path)),
		})
	}
	return usages
}

// SummarizeLibraries derives configured library_modules diagnostics.
func SummarizeLibraries(cfg config.Config, result *Result) *LibrarySummary {
	libraryModules := cfg.LibraryModules()
	if !cfg.Present() || libraryModules == nil || len(libraryModules.Paths()) == 0 || result == nil || result.Graph == nil {
		return nil
	}
	orphans := make([]string, 0)
	for _, module := range result.Libraries.Modules {
		if !result.Graph.HasLibraryConsumers(module.Path) {
			orphans = append(orphans, module.RelativePath)
		}
	}
	return &LibrarySummary{
		ConfiguredPaths: len(libraryModules.Paths()),
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
