package workflow

import (
	"context"
	"errors"
	"fmt"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
)

// TargetSelectionOptions controls how executable targets are selected from a workflow result.
type TargetSelectionOptions struct {
	ModulePath  string
	ChangedOnly bool
	BaseRef     string
	Filters     *filter.Flags
}

type changeDetectorResolver func() (plugin.ChangeDetectionProvider, error)

// ResolveTargets applies module/path filters and optional change detection to a workflow result.
func ResolveTargets(ctx context.Context, appCtx *plugin.AppContext, result *Result, opts TargetSelectionOptions) ([]*discovery.Module, error) {
	return resolveTargets(ctx, appCtx, result, opts, registry.ResolveChangeDetector)
}

func resolveTargets(
	ctx context.Context,
	appCtx *plugin.AppContext,
	result *Result,
	opts TargetSelectionOptions,
	resolveChangeDetector changeDetectorResolver,
) ([]*discovery.Module, error) {
	if appCtx == nil {
		return nil, errors.New("app context is required")
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

	detector, err := resolveChangeDetector()
	if err != nil {
		return nil, fmt.Errorf("change detection: %w", err)
	}

	cfg := appCtx.Config()
	changedModules, _, err := detector.DetectChangedModules(ctx, appCtx, opts.BaseRef, result.FullIndex)
	if err != nil {
		return nil, fmt.Errorf("detect changed modules: %w", err)
	}

	changedIDs := moduleIDs(changedModules)
	var affectedIDs []string

	if cfg.LibraryModules != nil && len(cfg.LibraryModules.Paths) > 0 {
		libraryPaths, libraryErr := detector.DetectChangedLibraries(ctx, appCtx, opts.BaseRef, cfg.LibraryModules.Paths)
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

	targets = resolveAffectedModules(cfg, opts.Filters, affectedIDs, changedIDs, result.FullIndex, result.FilteredIndex)
	if opts.ModulePath != "" {
		targets = filterModulesByPath(targets, opts.ModulePath)
	}

	return targets, nil
}

func resolveAffectedModules(
	cfg *config.Config,
	ff *filter.Flags,
	affectedIDs, changedIDs []string,
	fullIndex, filteredIndex *discovery.ModuleIndex,
) []*discovery.Module {
	idSet := make(map[string]bool, len(affectedIDs)+len(changedIDs))
	for _, id := range affectedIDs {
		idSet[id] = true
	}
	for _, id := range changedIDs {
		idSet[id] = true
	}

	targets := make([]*discovery.Module, 0, len(idSet))
	for id := range idSet {
		if module := filteredIndex.ByID(id); module != nil {
			targets = append(targets, module)
			continue
		}

		module := fullIndex.ByID(id)
		if module == nil {
			continue
		}
		if filtered := ApplyFilters(cfg, ff, []*discovery.Module{module}); len(filtered) > 0 {
			targets = append(targets, module)
		}
	}

	return targets
}

func filterModulesByPath(modules []*discovery.Module, modulePath string) []*discovery.Module {
	if modulePath == "" {
		return modules
	}
	filtered := modules[:0]
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
