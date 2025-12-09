package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/filter"
	"github.com/edelwud/terraci/internal/graph"
	"github.com/edelwud/terraci/internal/parser"
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

	fmt.Println("Validating Terraform project structure...")
	fmt.Println()

	// 1. Discover modules
	fmt.Printf("Scanning: %s\n", workDir)
	scanner := discovery.NewScanner(workDir)
	scanner.MinDepth = cfg.Structure.MinDepth
	scanner.MaxDepth = cfg.Structure.MaxDepth

	modules, err := scanner.Scan()
	if err != nil {
		return fmt.Errorf("failed to scan modules: %w", err)
	}

	fmt.Printf("  Found %d modules\n", len(modules))

	if len(modules) == 0 {
		fmt.Fprintln(os.Stderr, "ERROR: No modules found")
		return fmt.Errorf("no modules found")
	}

	// 2. Apply filters
	allExcludes := append([]string{}, cfg.Exclude...)
	allExcludes = append(allExcludes, excludes...)
	allIncludes := append([]string{}, cfg.Include...)
	allIncludes = append(allIncludes, includes...)
	globFilter := filter.NewGlobFilter(allExcludes, allIncludes)
	filteredModules := globFilter.FilterModules(modules)

	if len(filteredModules) != len(modules) {
		fmt.Printf("  After filtering: %d modules\n", len(filteredModules))
	}
	fmt.Println()

	// 3. Build module index
	moduleIndex := discovery.NewModuleIndex(filteredModules)

	// 4. Parse dependencies
	fmt.Println("Parsing dependencies...")

	hclParser := parser.NewParser()
	depExtractor := parser.NewDependencyExtractor(hclParser, moduleIndex)

	deps, errs := depExtractor.ExtractAllDependencies()

	if len(errs) > 0 {
		fmt.Printf("  Warnings: %d\n", len(errs))
		if verbose {
			for _, e := range errs {
				fmt.Fprintf(os.Stderr, "    WARN: %s\n", e)
			}
		}
	}

	// Count dependencies
	totalDeps := 0
	for _, d := range deps {
		totalDeps += len(d.DependsOn)
	}
	fmt.Printf("  Total dependency links: %d\n", totalDeps)
	fmt.Println()

	// 5. Build and validate dependency graph
	fmt.Println("Validating dependency graph...")

	depGraph := graph.BuildFromDependencies(filteredModules, deps)

	// Check for cycles
	cycles := depGraph.DetectCycles()
	if len(cycles) > 0 {
		hasErrors = true
		fmt.Printf("  ERROR: Circular dependencies detected: %d\n", len(cycles))
		for i, cycle := range cycles {
			fmt.Printf("    Cycle %d: %v\n", i+1, cycle)
		}
	} else {
		fmt.Println("  No circular dependencies")
	}

	// Get stats
	stats := depGraph.GetStats()
	fmt.Printf("  Root modules (no deps): %d\n", stats.RootModules)
	fmt.Printf("  Leaf modules (no dependents): %d\n", stats.LeafModules)
	fmt.Printf("  Max dependency depth: %d\n", stats.MaxDepth)
	fmt.Println()

	// 6. Verify execution order
	fmt.Println("Checking execution order...")

	levels, err := depGraph.ExecutionLevels()
	if err != nil {
		hasErrors = true
		fmt.Printf("  ERROR: Cannot determine execution order: %s\n", err)
	} else {
		fmt.Printf("  Execution levels: %d\n", len(levels))
		if verbose {
			for i, level := range levels {
				fmt.Printf("    Level %d: %d modules\n", i, len(level))
			}
		}
	}
	fmt.Println()

	// 7. Summary
	if hasErrors {
		fmt.Println("Validation FAILED - please fix the issues above")
		return fmt.Errorf("validation failed")
	}

	fmt.Println("Validation PASSED")
	return nil
}
