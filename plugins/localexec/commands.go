package localexec

import (
	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/plugin"
)

type sharedFlags struct {
	changedOnly bool
	baseRef     string
	modulePath  string
	parallelism int
	dryRun      bool
	filters     filter.Flags
}

func (sf *sharedFlags) toRequest(mode ExecutionMode) ExecuteRequest {
	return ExecuteRequest{
		ChangedOnly: sf.changedOnly,
		BaseRef:     sf.baseRef,
		Mode:        mode,
		ModulePath:  sf.modulePath,
		Parallelism: sf.parallelism,
		DryRun:      sf.dryRun,
		Filters:     &sf.filters,
	}
}

func (p *Plugin) Commands(appCtx *plugin.AppContext) []*cobra.Command {
	executor := NewExecutor(appCtx)

	cmd := &cobra.Command{
		Use:   "local-exec",
		Short: "Execute the generated terraci flow locally",
	}

	cmd.AddCommand(
		newPlanCmd(executor),
		newApplyCmd(executor),
		newRunCmd(executor),
	)

	return []*cobra.Command{cmd}
}

func newPlanCmd(executor Executor) *cobra.Command {
	var sf sharedFlags
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Run plan stages only (pre-plan, plan, post-plan)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return executor.Run(cmd.Context(), sf.toRequest(ExecutionModePlanOnly))
		},
	}
	registerSharedFlags(cmd, &sf)
	return cmd
}

func newApplyCmd(executor Executor) *cobra.Command {
	var sf sharedFlags
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Run apply stages only (pre-apply, apply, post-apply)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return executor.Run(cmd.Context(), sf.toRequest(ExecutionModeApplyOnly))
		},
	}
	registerSharedFlags(cmd, &sf)
	return cmd
}

func newRunCmd(executor Executor) *cobra.Command {
	var sf sharedFlags
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run all stages (plan + apply)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return executor.Run(cmd.Context(), sf.toRequest(ExecutionModeFull))
		},
	}
	registerSharedFlags(cmd, &sf)
	return cmd
}

func registerSharedFlags(cmd *cobra.Command, sf *sharedFlags) {
	cmd.Flags().BoolVar(&sf.changedOnly, "changed-only", false, "only include changed modules and their dependents")
	cmd.Flags().StringVar(&sf.baseRef, "base-ref", "", "base git ref for change detection")
	cmd.Flags().StringVarP(&sf.modulePath, "module", "m", "", "restrict execution to a single module path")
	cmd.Flags().IntVar(&sf.parallelism, "parallelism", 0, "override local execution parallelism")
	cmd.Flags().BoolVar(&sf.dryRun, "dry-run", false, "print the local execution order without running commands")
	cmd.Flags().StringArrayVarP(&sf.filters.Excludes, "exclude", "x", nil, "glob patterns to exclude modules")
	cmd.Flags().StringArrayVarP(&sf.filters.Includes, "include", "i", nil, "glob patterns to include modules")
	cmd.Flags().StringArrayVarP(&sf.filters.SegmentArgs, "filter", "f", nil, "filter by segment (e.g. -f environment=stage)")
}
