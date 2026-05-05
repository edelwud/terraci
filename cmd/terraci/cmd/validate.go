package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/workflow"
)

func newValidateCmd(app *App) *cobra.Command {
	ff := &filter.Flags{}

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate module structure and dependencies",
		// validate is a read-only command — it doesn't drive any plugin runtime
		// (cost engines, policy clients, change detectors). Skip preflight so a
		// missing .git directory or a misconfigured cache backend doesn't
		// derail what is fundamentally a graph-correctness check.
		Annotations: map[string]string{annotationSkipPreflight: annotationTrue},
		Long: `Validate the Terraform module structure and check for dependency issues.

This command will:
  - Scan for modules following the expected directory structure
  - Parse terraform_remote_state references
  - Build the dependency graph
  - Check for circular dependencies
  - Report any issues found`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			hasErrors := false

			log.Info("validating terraform project structure")

			result, err := workflow.Run(cmd.Context(), workflowOptions(app, ff))
			if err != nil {
				return err
			}

			if len(result.Warnings) > 0 {
				log.WithField("count", len(result.Warnings)).Warn("warnings during parsing")
				log.IncreasePadding()
				for _, e := range result.Warnings {
					log.WithField("warning", e.Error()).Debug("parser warning")
				}
				log.DecreasePadding()
			}

			// Count dependencies
			totalDeps := 0
			for _, d := range result.Dependencies {
				totalDeps += len(d.DependsOn)
			}
			log.WithField("count", totalDeps).Info("dependency links found")

			log.Info("validating dependency graph")

			depGraph := result.Graph

			// Check for cycles
			cycles := depGraph.DetectCycles()
			if len(cycles) > 0 {
				hasErrors = true
				log.WithField("count", len(cycles)).Error("circular dependencies detected")
				log.IncreasePadding()
				for i, cycle := range cycles {
					log.WithField("cycle", i+1).WithField("path", fmt.Sprintf("%v", cycle)).Error("cycle")
				}
				log.DecreasePadding()
			} else {
				log.Info("no circular dependencies")
			}

			// Get stats
			stats := depGraph.GetStats()
			log.IncreasePadding()
			log.WithField("count", stats.RootModules).Debug("root modules (no deps)")
			log.WithField("count", stats.LeafModules).Debug("leaf modules (no dependents)")
			log.WithField("depth", stats.MaxDepth).Debug("max dependency depth")
			log.DecreasePadding()

			// 6. Verify execution order
			log.Info("checking execution order")

			levels, err := depGraph.ExecutionLevels()
			if err != nil {
				hasErrors = true
				log.WithError(err).Error("cannot determine execution order")
			} else {
				log.WithField("levels", len(levels)).Info("execution levels determined")
				if isDebug() {
					log.IncreasePadding()
					for i, level := range levels {
						log.WithField("level", i).WithField("modules", len(level)).Debug("level")
					}
					log.DecreasePadding()
				}
			}

			// 7. Summary
			if hasErrors {
				log.Error("validation FAILED - please fix the issues above")
				return errors.New("validation failed")
			}

			log.Info("validation PASSED")
			return nil
		},
	}

	registerFilterFlags(cmd, ff)

	return cmd
}
