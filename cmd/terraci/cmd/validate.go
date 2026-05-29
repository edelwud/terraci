package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/cmd/terraci/internal/projectflow"
	"github.com/edelwud/terraci/cmd/terraci/internal/runflow"
	"github.com/edelwud/terraci/cmd/terraci/internal/validateflow"
	"github.com/edelwud/terraci/pkg/diagnostic"
	"github.com/edelwud/terraci/pkg/filter"
)

func newValidateCmd() *cobra.Command {
	ff := &filter.Flags{}

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate module structure and dependencies",
		// validate is a read-only command — it doesn't drive any plugin runtime
		// (cost engines, policy clients, change detectors). Skip preflight so a
		// missing .git directory or a misconfigured cache backend doesn't
		// derail what is fundamentally a graph-correctness check.
		Long: `Validate the Terraform module structure and check for dependency issues.

This command will:
  - Scan for modules following the expected directory structure
  - Parse terraform_remote_state references
  - Build the dependency graph
  - Check for circular dependencies
  - Report any issues found`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			prepared, err := runflow.FromContext(cmd.Context())
			if err != nil {
				return err
			}

			log.Info("validating terraform project structure")

			result, err := validateflow.Run(cmd.Context(), validateflow.NewRuntime(prepared), validateflow.Request{
				Filters: *ff,
			})
			if err != nil {
				return err
			}

			logValidationResult(result)
			log.Info("validating dependency graph")
			logGraphValidation(result)
			logExecutionOrder(result)
			reportLibraryModules(result.Project.LibrarySummary)

			if !result.Passed {
				log.Error("validation FAILED - please fix the issues above")
				return errors.New("validation failed")
			}

			log.Info("validation PASSED")
			return nil
		},
	}
	runflow.MarkCommand(cmd, runflow.CommandPolicy{SkipPreflight: true})

	registerFilterFlags(cmd, ff)

	return cmd
}

func logValidationResult(result *validateflow.Result) {
	warnings := result.Project.Workflow.Diagnostics.Filter(diagnostic.SeverityWarning)
	if len(warnings) > 0 {
		log.WithField("count", len(warnings)).Warn("warnings during parsing")
		log.IncreasePadding()
		for _, diag := range warnings {
			log.WithField("warning", diag.Message()).Debug("parser warning")
		}
		log.DecreasePadding()
	}
	log.WithField("count", result.DependencyLinks).Info("dependency links found")
}

func logGraphValidation(result *validateflow.Result) {
	if len(result.Cycles) > 0 {
		log.WithField("count", len(result.Cycles)).Error("circular dependencies detected")
		log.IncreasePadding()
		for i, cycle := range result.Cycles {
			log.WithField("cycle", i+1).WithField("path", fmt.Sprintf("%v", cycle)).Error("cycle")
		}
		log.DecreasePadding()
	} else {
		log.Info("no circular dependencies")
	}

	log.IncreasePadding()
	log.WithField("count", result.Stats.RootModules).Debug("root modules (no deps)")
	log.WithField("count", result.Stats.LeafModules).Debug("leaf modules (no dependents)")
	log.WithField("depth", result.Stats.MaxDepth).Debug("max dependency depth")
	log.DecreasePadding()
}

func logExecutionOrder(result *validateflow.Result) {
	log.Info("checking execution order")
	if result.ExecutionLevelsError != nil {
		log.WithError(result.ExecutionLevelsError).Error("cannot determine execution order")
		return
	}
	log.WithField("levels", len(result.ExecutionLevels)).Info("execution levels determined")
	if isDebug() {
		log.IncreasePadding()
		for i, level := range result.ExecutionLevels {
			log.WithField("level", i).WithField("modules", len(level)).Debug("level")
		}
		log.DecreasePadding()
	}
}

func reportLibraryModules(summary *projectflow.LibrarySummary) {
	if summary == nil {
		return
	}

	log.Info("library modules")
	log.IncreasePadding()
	log.WithField("paths", summary.ConfiguredPaths).Info("configured library roots")
	log.WithField("count", summary.Discovered).Info("discovered library modules")
	log.WithField("count", summary.Consumers).Info("executable modules using libraries")
	if len(summary.Orphans) > 0 {
		log.WithField("count", len(summary.Orphans)).Warn("orphan library modules (no executable consumers)")
		log.IncreasePadding()
		for _, id := range summary.Orphans {
			log.WithField("module", id).Warn("orphan library")
		}
		log.DecreasePadding()
	}
	log.DecreasePadding()
}
