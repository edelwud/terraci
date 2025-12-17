package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/filter"
	"github.com/edelwud/terraci/internal/graph"
	"github.com/edelwud/terraci/internal/parser"
	"github.com/edelwud/terraci/pkg/log"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate module structure and dependencies",
	Long: `Validate the Terraform module structure and check for dependency issues.

This command will:
  - Scan for modules following the expected directory structure
  - Parse terraform_remote_state references
  - Build the dependency graph
  - Check for circular dependencies
  - Report any issues found`,
	RunE: runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().StringArrayVarP(&excludes, "exclude", "x", nil, "glob patterns to exclude modules")
	validateCmd.Flags().StringArrayVarP(&includes, "include", "i", nil, "glob patterns to include modules")
}

func runValidate(_ *cobra.Command, _ []string) error {
	hasErrors := false

	log.Info("validating terraform project structure")

	// 1. Discover modules
	log.WithField("dir", workDir).Info("scanning for modules")
	scanner := discovery.NewScanner(workDir)
	scanner.MinDepth = cfg.Structure.MinDepth
	scanner.MaxDepth = cfg.Structure.MaxDepth

	modules, err := scanner.Scan()
	if err != nil {
		return fmt.Errorf("failed to scan modules: %w", err)
	}

	log.WithField("count", len(modules)).Info("modules found")

	if len(modules) == 0 {
		log.Error("no modules found")
		return fmt.Errorf("no modules found")
	}

	// 2. Apply filters
	allExcludes := append([]string{}, cfg.Exclude...)
	allExcludes = append(allExcludes, excludes...)
	allIncludes := append([]string{}, cfg.Include...)
	allIncludes = append(allIncludes, includes...)
	filteredModules := filter.Apply(modules, filter.Options{
		Excludes: allExcludes,
		Includes: allIncludes,
	})

	if len(filteredModules) != len(modules) {
		log.WithField("count", len(filteredModules)).Info("modules after filtering")
	}

	// 3. Build module index
	moduleIndex := discovery.NewModuleIndex(filteredModules)

	// 4. Parse dependencies
	log.Info("parsing dependencies")

	hclParser := parser.NewParser()
	depExtractor := parser.NewDependencyExtractor(hclParser, moduleIndex)

	deps, errs := depExtractor.ExtractAllDependencies()

	if len(errs) > 0 {
		log.WithField("count", len(errs)).Warn("warnings during parsing")
		log.IncreasePadding()
		for _, e := range errs {
			log.WithField("warning", e.Error()).Debug("parser warning")
		}
		log.DecreasePadding()
	}

	// Count dependencies
	totalDeps := 0
	for _, d := range deps {
		totalDeps += len(d.DependsOn)
	}
	log.WithField("count", totalDeps).Info("dependency links found")

	// 5. Build and validate dependency graph
	log.Info("validating dependency graph")

	depGraph := graph.BuildFromDependencies(filteredModules, deps)

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
		if log.IsDebug() {
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
		return fmt.Errorf("validation failed")
	}

	log.Info("validation PASSED")
	return nil
}
