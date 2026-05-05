package workflow

import (
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/filter"
)

// MergedFilterOptions merges config defaults with CLI filter flags.
func MergedFilterOptions(cfg *config.Config, ff *filter.Flags) filter.Options {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	return ff.Merge(cfg.Exclude, cfg.Include)
}

// OptionsFromConfig builds workflow options from configuration and CLI filters.
func OptionsFromConfig(workDir string, cfg *config.Config, ff *filter.Flags) Options {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	opts := MergedFilterOptions(cfg, ff)
	return Options{
		WorkDir:        workDir,
		Segments:       cfg.Structure.Segments,
		Excludes:       opts.Excludes,
		Includes:       opts.Includes,
		SegmentFilters: opts.Segments,
		LibraryPaths:   libraryPathsFromConfig(cfg),
	}
}

// libraryPathsFromConfig returns the configured library_modules.paths slice or
// nil when unset. Returning the slice as-is is intentional: scanner cleans it.
func libraryPathsFromConfig(cfg *config.Config) []string {
	if cfg == nil || cfg.LibraryModules == nil {
		return nil
	}
	return cfg.LibraryModules.Paths
}

// ApplyFilters applies config and CLI filters to a module list.
func ApplyFilters(cfg *config.Config, ff *filter.Flags, modules []*discovery.Module) []*discovery.Module {
	return filter.Apply(modules, MergedFilterOptions(cfg, ff))
}
