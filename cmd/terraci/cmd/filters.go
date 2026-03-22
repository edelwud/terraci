package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/filter"
	"github.com/edelwud/terraci/internal/workflow"
)

// filterFlags holds shared filter flag values.
type filterFlags struct {
	excludes    []string
	includes    []string
	segmentArgs []string
}

// registerFilterFlags adds common filter flags to a command.
func (f *filterFlags) register(cmd *cobra.Command) {
	cmd.Flags().StringArrayVarP(&f.excludes, "exclude", "x", nil, "glob patterns to exclude modules")
	cmd.Flags().StringArrayVarP(&f.includes, "include", "i", nil, "glob patterns to include modules")
	cmd.Flags().StringArrayVarP(&f.segmentArgs, "filter", "f", nil, "filter by segment (e.g. -f environment=stage)")
}

// mergedExcludes returns config + CLI exclude patterns.
func (f *filterFlags) mergedExcludes(app *App) []string {
	return append(append([]string{}, app.Config.Exclude...), f.excludes...)
}

// mergedIncludes returns config + CLI include patterns.
func (f *filterFlags) mergedIncludes(app *App) []string {
	return append(append([]string{}, app.Config.Include...), f.includes...)
}

// parsedSegmentFilters parses --filter key=value flags into a map.
func (f *filterFlags) parsedSegmentFilters() map[string][]string {
	segments := make(map[string][]string)
	for _, arg := range f.segmentArgs {
		if k, v, ok := strings.Cut(arg, "="); ok && k != "" {
			segments[k] = append(segments[k], v)
		}
	}
	return segments
}

// applyFilters applies config + CLI filters to modules.
func (f *filterFlags) applyFilters(app *App, modules []*discovery.Module) []*discovery.Module {
	return filter.Apply(modules, filter.Options{
		Excludes: f.mergedExcludes(app),
		Includes: f.mergedIncludes(app),
		Segments: f.parsedSegmentFilters(),
	})
}

// workflowOptions returns workflow.Options from app config and filter flags.
func (f *filterFlags) workflowOptions(app *App) workflow.Options {
	return workflow.Options{
		WorkDir:        app.WorkDir,
		Segments:       app.Config.Structure.Segments,
		Excludes:       f.mergedExcludes(app),
		Includes:       f.mergedIncludes(app),
		SegmentFilters: f.parsedSegmentFilters(),
	}
}
