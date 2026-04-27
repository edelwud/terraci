package workflow

import (
	"context"
	"errors"
	"fmt"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/filter"
)

// TargetSelectionOptions controls how executable targets are selected from a workflow result.
type TargetSelectionOptions struct {
	ModulePath  string
	ChangedOnly bool
	BaseRef     string
	Filters     *filter.Flags

	ChangeDetectorResolver ChangeDetectorResolver
}

// ChangeDetectorResolver resolves the change detection provider for changed-only target selection.
type ChangeDetectorResolver func() (ChangeDetector, error)

// ChangeDetector detects changed modules and library paths for target selection.
type ChangeDetector interface {
	DetectChangedModules(ctx context.Context, workDir, baseRef string, moduleIndex *discovery.ModuleIndex) (changed []*discovery.Module, changedFiles []string, err error)
	DetectChangedLibraries(ctx context.Context, workDir, baseRef string, libraryPaths []string) ([]string, error)
}

// ResolveTargets applies module/path filters and optional change detection to a workflow result.
func ResolveTargets(ctx context.Context, workDir string, cfg *config.Config, result *Result, opts TargetSelectionOptions) ([]*discovery.Module, error) {
	return resolveTargets(ctx, workDir, cfg, result, opts)
}

func resolveTargets(
	ctx context.Context,
	workDir string,
	cfg *config.Config,
	result *Result,
	opts TargetSelectionOptions,
) ([]*discovery.Module, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}
	if result == nil {
		return nil, errors.New("workflow result is required")
	}

	targets := result.FilteredModules
	if opts.ModulePath != "" {
		targets = filterModulesByPath(targets, opts.ModulePath)
	}
	if !opts.ChangedOnly {
		if len(targets) == 0 {
			return nil, errors.New("no modules remaining after filtering")
		}
		return targets, nil
	}

	if opts.ChangeDetectorResolver == nil {
		return nil, errors.New("change detector resolver is required for changed-only target selection")
	}

	detector, err := opts.ChangeDetectorResolver()
	if err != nil {
		return nil, fmt.Errorf("change detection: %w", err)
	}

	changedModules, _, err := detector.DetectChangedModules(ctx, workDir, opts.BaseRef, result.FullIndex)
	if err != nil {
		return nil, fmt.Errorf("detect changed modules: %w", err)
	}

	changedIDs := moduleIDs(changedModules)
	var affectedIDs []string

	if cfg.LibraryModules != nil && len(cfg.LibraryModules.Paths) > 0 {
		libraryPaths, libraryErr := detector.DetectChangedLibraries(ctx, workDir, opts.BaseRef, cfg.LibraryModules.Paths)
		if libraryErr != nil {
			return nil, fmt.Errorf("detect changed libraries: %w", libraryErr)
		}
		if len(libraryPaths) > 0 {
			affectedIDs = result.Graph.GetAffectedModulesWithLibraries(changedIDs, libraryPaths)
		}
	}
	if len(affectedIDs) == 0 {
		affectedIDs = result.Graph.GetAffectedModules(changedIDs)
	}

	targets = resolveAffectedModules(cfg, opts.Filters, affectedIDs, changedIDs, result.AllModules, result.FilteredModules, result.FullIndex, result.FilteredIndex)
	if opts.ModulePath != "" {
		targets = filterModulesByPath(targets, opts.ModulePath)
	}

	return targets, nil
}

func resolveAffectedModules(
	cfg *config.Config,
	ff *filter.Flags,
	affectedIDs, changedIDs []string,
	allModules, filteredModules []*discovery.Module,
	fullIndex, filteredIndex *discovery.ModuleIndex,
) []*discovery.Module {
	if len(allModules) == 0 && fullIndex != nil {
		allModules = fullIndex.All()
	}
	if len(filteredModules) == 0 && filteredIndex != nil {
		filteredModules = filteredIndex.All()
	}

	idSet := make(map[string]bool, len(affectedIDs)+len(changedIDs))
	for _, id := range affectedIDs {
		idSet[id] = true
	}
	for _, id := range changedIDs {
		idSet[id] = true
	}

	targets := make([]*discovery.Module, 0, len(idSet))
	seen := make(map[string]bool, len(idSet))

	for _, module := range filteredModules {
		id := module.ID()
		if idSet[id] {
			targets = append(targets, module)
			seen[id] = true
		}
	}

	for _, module := range allModules {
		id := module.ID()
		if !idSet[id] || seen[id] {
			continue
		}
		if filteredIndex.ByID(id) != nil {
			continue
		}
		if fullIndex.ByID(id) != nil && len(ApplyFilters(cfg, ff, []*discovery.Module{module})) > 0 {
			targets = append(targets, module)
			seen[id] = true
		}
	}

	return targets
}

func filterModulesByPath(modules []*discovery.Module, modulePath string) []*discovery.Module {
	if modulePath == "" {
		return modules
	}
	filtered := make([]*discovery.Module, 0, len(modules))
	for _, module := range modules {
		if module.RelativePath == modulePath {
			filtered = append(filtered, module)
		}
	}
	return filtered
}

func moduleIDs(modules []*discovery.Module) []string {
	ids := make([]string, len(modules))
	for i := range modules {
		ids[i] = modules[i].ID()
	}
	return ids
}
