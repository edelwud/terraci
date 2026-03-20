package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/filter"
)

// Shared filter flags — reused across generate, graph, validate commands.
var (
	excludes    []string
	includes    []string
	segmentArgs []string // --filter key=value pairs
)

// registerFilterFlags adds common filter flags to a command.
func registerFilterFlags(cmd *cobra.Command) {
	cmd.Flags().StringArrayVarP(&excludes, "exclude", "x", nil, "glob patterns to exclude modules")
	cmd.Flags().StringArrayVarP(&includes, "include", "i", nil, "glob patterns to include modules")
	cmd.Flags().StringArrayVarP(&segmentArgs, "filter", "f", nil, "filter by segment (e.g. -f environment=stage)")
}

// applyFilters applies config + CLI filters to modules.
func applyFilters(modules []*discovery.Module) []*discovery.Module {
	allExcludes := append(append([]string{}, cfg.Exclude...), excludes...)
	allIncludes := append(append([]string{}, cfg.Include...), includes...)

	segments := make(map[string][]string)
	for _, arg := range segmentArgs {
		if k, v, ok := strings.Cut(arg, "="); ok && k != "" {
			segments[k] = append(segments[k], v)
		}
	}

	return filter.Apply(modules, filter.Options{
		Excludes: allExcludes,
		Includes: allIncludes,
		Segments: segments,
	})
}
