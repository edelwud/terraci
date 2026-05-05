package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/config"
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

			reportLibraryModules(app, result)

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

// libraryModulesSummary captures the diagnostic data validate prints about
// library modules. It is computed in a pure function so tests can verify
// orphan detection without poking at logger output.
type libraryModulesSummary struct {
	ConfiguredPaths int
	Discovered      int
	Consumers       int
	Orphans         []string
}

// computeLibraryModulesSummary derives a libraryModulesSummary from the
// validated workflow result. Returns nil when no library paths are configured.
func computeLibraryModulesSummary(cfg *config.Config, result *workflow.Result) *libraryModulesSummary {
	if cfg == nil || cfg.LibraryModules == nil || len(cfg.LibraryModules.Paths) == 0 {
		return nil
	}
	libraries := result.Libraries.Modules
	orphans := make([]string, 0)
	for _, m := range libraries {
		if !result.Graph.HasLibraryConsumers(m.Path) {
			orphans = append(orphans, m.RelativePath)
		}
	}
	return &libraryModulesSummary{
		ConfiguredPaths: len(cfg.LibraryModules.Paths),
		Discovered:      len(libraries),
		Consumers:       result.Graph.LibraryConsumerCount(),
		Orphans:         orphans,
	}
}

// reportLibraryModules logs a summary of configured library_modules using
// computeLibraryModulesSummary. No-op when the feature is not configured.
func reportLibraryModules(app *App, result *workflow.Result) {
	summary := computeLibraryModulesSummary(app.Config, result)
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
