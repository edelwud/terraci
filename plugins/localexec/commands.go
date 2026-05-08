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
	filters     filter.Flags
}

func (sf *sharedFlags) toRequest(mode ExecutionMode) ExecuteRequest {
	return ExecuteRequest{
		ChangedOnly: sf.changedOnly,
		BaseRef:     sf.baseRef,
		Mode:        mode,
		ModulePath:  sf.modulePath,
		Parallelism: sf.parallelism,
		Filters:     &sf.filters,
	}
}

func (p *Plugin) Commands() []*cobra.Command {
	cmd := &cobra.Command{
		Use:   "local-exec",
		Short: "Execute the generated terraci flow locally",
		Long: `Execute the terraci pipeline IR locally against the current Terraform project.

Use "plan" to run plan jobs and standalone jobs whose resource inputs are available.
Use "run" to run the full local flow: plan, apply, and resource-dependent jobs.
After execution, local-exec always prints a local DAG/job summary.

Target selection flags such as --module, --filter, --include, --exclude, and
--changed-only are available on the "plan" and "run" subcommands. If no modules
match, the command exits cleanly after logging "no modules to process".`,
		Example: `  terraci local-exec plan
  terraci local-exec plan --changed-only
  terraci local-exec plan --filter environment=stage
  terraci local-exec run --changed-only
  terraci local-exec plan --module platform/stage/eu-central-1/vpc
  terraci local-exec run --filter environment=stage --parallelism 2`,
	}

	cmd.AddCommand(
		newPlanCmd(),
		newRunCmd(),
	)

	return []*cobra.Command{cmd}
}

func newPlanCmd() *cobra.Command {
	var sf sharedFlags
	cmd := &cobra.Command{
		Use:   cmdPlan,
		Short: "Run the plan DAG locally",
		Long: `Run local planning for the selected modules and then execute standalone
jobs whose resource inputs are available in plan mode. local-exec always prints
the execution summary. If target selection resolves to no modules, the command
exits without error after logging "no modules to process".`,
		Example: `  terraci local-exec plan
  terraci local-exec plan --changed-only
  terraci local-exec plan --module platform/stage/eu-central-1/vpc
  terraci local-exec plan --filter environment=stage
  terraci local-exec plan --include 'platform/*' --exclude '*/test/*'`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			appCtx := plugin.FromContext(cmd.Context())
			return NewExecutor(appCtx).Run(cmd.Context(), sf.toRequest(ExecutionModePlan))
		},
	}
	registerSharedFlags(cmd, &sf)
	return cmd
}

func newRunCmd() *cobra.Command {
	var sf sharedFlags
	cmd := &cobra.Command{
		Use:   cmdRun,
		Short: "Run the full DAG locally",
		Long: `Run the full local execution flow for the selected modules: plan, apply,
and standalone resource-dependent jobs. local-exec always prints the execution
summary. If target selection resolves to no modules, the command exits without
error after logging "no modules to process".`,
		Example: `  terraci local-exec run
  terraci local-exec run --changed-only
  terraci local-exec run --module platform/stage/eu-central-1/vpc
  terraci local-exec run --filter environment=stage --parallelism 2`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			appCtx := plugin.FromContext(cmd.Context())
			return NewExecutor(appCtx).Run(cmd.Context(), sf.toRequest(ExecutionModeRun))
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
	cmd.Flags().StringArrayVarP(&sf.filters.Excludes, "exclude", "x", nil, "glob patterns to exclude modules")
	cmd.Flags().StringArrayVarP(&sf.filters.Includes, "include", "i", nil, "glob patterns to include modules")
	cmd.Flags().StringArrayVarP(&sf.filters.SegmentArgs, "filter", "f", nil, "filter by segment (e.g. -f environment=stage)")
}
