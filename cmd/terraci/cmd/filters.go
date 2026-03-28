package cmd

import (
	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/workflow"
)

// registerFilterFlags adds common filter flags to a cobra command.
func registerFilterFlags(cmd *cobra.Command, f *filter.Flags) {
	cmd.Flags().StringArrayVarP(&f.Excludes, "exclude", "x", nil, "glob patterns to exclude modules")
	cmd.Flags().StringArrayVarP(&f.Includes, "include", "i", nil, "glob patterns to include modules")
	cmd.Flags().StringArrayVarP(&f.SegmentArgs, "filter", "f", nil, "filter by segment (e.g. -f environment=stage)")
}

// mergedFilterOpts merges CLI filter flags with config-level exclude/include.
func mergedFilterOpts(app *App, ff *filter.Flags) filter.Options {
	return ff.Merge(app.Config.Exclude, app.Config.Include)
}

// workflowOptions builds workflow.Options from app config and filter flags.
func workflowOptions(app *App, ff *filter.Flags) workflow.Options {
	opts := mergedFilterOpts(app, ff)
	return workflow.Options{
		WorkDir:        app.WorkDir,
		Segments:       app.Config.Structure.Segments,
		Excludes:       opts.Excludes,
		Includes:       opts.Includes,
		SegmentFilters: opts.Segments,
	}
}

// applyFilters applies config + CLI filters to a module list.
func applyFilters(app *App, ff *filter.Flags, modules []*discovery.Module) []*discovery.Module {
	return filter.Apply(modules, mergedFilterOpts(app, ff))
}
