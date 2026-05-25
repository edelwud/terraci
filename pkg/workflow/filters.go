package workflow

import (
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/filter"
)

func mergedFilterOptions(cfg config.Snapshot, ff *filter.Flags) filter.Options {
	if !cfg.Present() {
		cfg = config.DefaultConfig().Snapshot()
	}
	return ff.Merge(cfg.Exclude(), cfg.Include())
}

func optionsFromConfig(workDir string, cfg config.Snapshot, ff *filter.Flags) Options {
	if !cfg.Present() {
		cfg = config.DefaultConfig().Snapshot()
	}
	opts := mergedFilterOptions(cfg, ff)
	structure := cfg.Structure()
	return Options{
		WorkDir:        workDir,
		Segments:       structure.Segments,
		Excludes:       opts.Excludes,
		Includes:       opts.Includes,
		SegmentFilters: opts.Segments,
		LibraryPaths:   libraryPathsFromConfig(cfg),
	}
}

// libraryPathsFromConfig returns the configured library_modules.paths slice or
// nil when unset. Returning the slice as-is is intentional: scanner cleans it.
func libraryPathsFromConfig(cfg config.Snapshot) []string {
	libraryModules := cfg.LibraryModules()
	if libraryModules == nil {
		return nil
	}
	return libraryModules.Paths
}

func applyFilters(cfg config.Snapshot, ff *filter.Flags, modules []*discovery.Module) ([]*discovery.Module, error) {
	return filter.Apply(modules, mergedFilterOptions(cfg, ff))
}
