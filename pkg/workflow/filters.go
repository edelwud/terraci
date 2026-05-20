package workflow

import (
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/filter"
)

// MergedFilterOptions merges config defaults with CLI filter flags.
func MergedFilterOptions(cfg config.Snapshot, ff *filter.Flags) filter.Options {
	if !cfg.Present() {
		cfg = config.DefaultConfig().Snapshot()
	}
	return ff.Merge(cfg.Exclude(), cfg.Include())
}

// OptionsFromConfig builds workflow options from configuration and CLI filters.
func OptionsFromConfig(workDir string, cfg config.Snapshot, ff *filter.Flags) Options {
	if !cfg.Present() {
		cfg = config.DefaultConfig().Snapshot()
	}
	opts := MergedFilterOptions(cfg, ff)
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

// ApplyFilters applies config and CLI filters to a module list.
func ApplyFilters(cfg config.Snapshot, ff *filter.Flags, modules []*discovery.Module) ([]*discovery.Module, error) {
	return filter.Apply(modules, MergedFilterOptions(cfg, ff))
}
