package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/terraci/terraci/internal/discovery"
	"github.com/terraci/terraci/internal/filter"
	"github.com/terraci/terraci/internal/git"
	"github.com/terraci/terraci/internal/graph"
	"github.com/terraci/terraci/internal/parser"
	"github.com/terraci/terraci/internal/pipeline/gitlab"
)

var (
	// Generate command flags
	outputFile   string
	changedOnly  bool
	baseRef      string
	excludes     []string
	includes     []string
	dryRun       bool
	services     []string
	environments []string
	regions      []string
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate GitLab CI pipeline",
	Long: `Generate a GitLab CI pipeline YAML file based on the Terraform
module structure and dependencies.

Examples:
  # Generate full pipeline
  terraci generate -o .gitlab-ci.yml

  # Generate pipeline only for changed modules
  terraci generate --changed-only --base-ref main

  # Generate with exclusions
  terraci generate --exclude "*/test/*" --exclude "cdp/*/eu-north-1/*"

  # Filter by environment
  terraci generate --environment stage --environment prod

  # Dry run to see what would be generated
  terraci generate --dry-run`,
	RunE: runGenerate,
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file (default: stdout)")
	generateCmd.Flags().BoolVar(&changedOnly, "changed-only", false, "only include changed modules and their dependents")
	generateCmd.Flags().StringVar(&baseRef, "base-ref", "", "base git ref for change detection (default: auto-detect)")
	generateCmd.Flags().StringArrayVarP(&excludes, "exclude", "x", nil, "glob patterns to exclude modules")
	generateCmd.Flags().StringArrayVarP(&includes, "include", "i", nil, "glob patterns to include modules")
	generateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be generated without creating output")
	generateCmd.Flags().StringArrayVarP(&services, "service", "s", nil, "filter by service name")
	generateCmd.Flags().StringArrayVarP(&environments, "environment", "e", nil, "filter by environment")
	generateCmd.Flags().StringArrayVarP(&regions, "region", "r", nil, "filter by region")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	// 1. Discover modules
	if verbose {
		fmt.Fprintf(os.Stderr, "Scanning directory: %s\n", workDir)
	}

	scanner := discovery.NewScanner(workDir)
	scanner.MinDepth = cfg.Structure.MinDepth
	scanner.MaxDepth = cfg.Structure.MaxDepth

	modules, err := scanner.Scan()
	if err != nil {
		return fmt.Errorf("failed to scan modules: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Found %d modules\n", len(modules))
	}

	if len(modules) == 0 {
		return fmt.Errorf("no modules found in %s", workDir)
	}

	// 2. Apply filters
	modules = applyFilters(modules)

	if verbose {
		fmt.Fprintf(os.Stderr, "After filtering: %d modules\n", len(modules))
	}

	if len(modules) == 0 {
		return fmt.Errorf("no modules remaining after filtering")
	}

	// 3. Build module index
	moduleIndex := discovery.NewModuleIndex(modules)

	// 4. Parse dependencies
	if verbose {
		fmt.Fprintf(os.Stderr, "Parsing dependencies...\n")
	}

	hclParser := parser.NewParser()
	depExtractor := parser.NewDependencyExtractor(hclParser, moduleIndex)

	deps, errs := depExtractor.ExtractAllDependencies()
	if verbose && len(errs) > 0 {
		fmt.Fprintf(os.Stderr, "Warnings during dependency extraction:\n")
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
	}

	// 5. Build dependency graph
	depGraph := graph.BuildFromDependencies(modules, deps)

	// Check for cycles
	cycles := depGraph.DetectCycles()
	if len(cycles) > 0 {
		fmt.Fprintf(os.Stderr, "WARNING: Circular dependencies detected:\n")
		for _, cycle := range cycles {
			fmt.Fprintf(os.Stderr, "  %v\n", cycle)
		}
	}

	// 6. Determine target modules
	targetModules := modules

	if changedOnly {
		changedModules, err := getChangedModules(moduleIndex)
		if err != nil {
			return fmt.Errorf("failed to detect changed modules: %w", err)
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "Changed modules: %d\n", len(changedModules))
			for _, m := range changedModules {
				fmt.Fprintf(os.Stderr, "  - %s\n", m.ID())
			}
		}

		// Get affected modules (changed + dependents)
		changedIDs := make([]string, len(changedModules))
		for i, m := range changedModules {
			changedIDs[i] = m.ID()
		}

		affectedIDs := depGraph.GetAffectedModules(changedIDs)

		targetModules = make([]*discovery.Module, 0, len(affectedIDs))
		for _, id := range affectedIDs {
			if m := moduleIndex.ByID(id); m != nil {
				targetModules = append(targetModules, m)
			}
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "Affected modules (including dependents): %d\n", len(targetModules))
		}
	}

	if len(targetModules) == 0 {
		fmt.Fprintln(os.Stderr, "No modules to process")
		return nil
	}

	// 7. Generate pipeline
	generator := gitlab.NewGenerator(cfg, depGraph, modules)

	// Handle dry run
	if dryRun {
		result, err := generator.DryRun(targetModules)
		if err != nil {
			return fmt.Errorf("dry run failed: %w", err)
		}

		fmt.Printf("Dry Run Results:\n")
		fmt.Printf("  Total modules discovered: %d\n", result.TotalModules)
		fmt.Printf("  Modules to process: %d\n", result.AffectedModules)
		fmt.Printf("  Pipeline stages: %d\n", result.Stages)
		fmt.Printf("  Pipeline jobs: %d\n", result.Jobs)
		fmt.Printf("\nExecution order:\n")
		for i, level := range result.ExecutionOrder {
			fmt.Printf("  Level %d: %v\n", i, level)
		}
		return nil
	}

	pipeline, err := generator.Generate(targetModules)
	if err != nil {
		return fmt.Errorf("failed to generate pipeline: %w", err)
	}

	// 8. Output
	yamlContent, err := pipeline.ToYAML()
	if err != nil {
		return fmt.Errorf("failed to serialize pipeline: %w", err)
	}

	// Add header comment
	header := []byte(`# Generated by terraci
# DO NOT EDIT - this file is auto-generated
# https://github.com/terraci/terraci

`)
	yamlContent = append(header, yamlContent...)

	if outputFile != "" {
		if err := os.WriteFile(outputFile, yamlContent, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Pipeline written to %s\n", outputFile)
	} else {
		fmt.Print(string(yamlContent))
	}

	return nil
}

func applyFilters(modules []*discovery.Module) []*discovery.Module {
	// Combine config excludes with command line excludes
	allExcludes := append(cfg.Exclude, excludes...)
	allIncludes := append(cfg.Include, includes...)

	// Apply glob filter
	globFilter := filter.NewGlobFilter(allExcludes, allIncludes)
	modules = globFilter.FilterModules(modules)

	// Apply service filter
	if len(services) > 0 {
		serviceFilter := &filter.ServiceFilter{Services: services}
		var filtered []*discovery.Module
		for _, m := range modules {
			if serviceFilter.Match(m) {
				filtered = append(filtered, m)
			}
		}
		modules = filtered
	}

	// Apply environment filter
	if len(environments) > 0 {
		envFilter := &filter.EnvironmentFilter{Environments: environments}
		var filtered []*discovery.Module
		for _, m := range modules {
			if envFilter.Match(m) {
				filtered = append(filtered, m)
			}
		}
		modules = filtered
	}

	// Apply region filter
	if len(regions) > 0 {
		regionFilter := &filter.RegionFilter{Regions: regions}
		var filtered []*discovery.Module
		for _, m := range modules {
			if regionFilter.Match(m) {
				filtered = append(filtered, m)
			}
		}
		modules = filtered
	}

	return modules
}

func getChangedModules(moduleIndex *discovery.ModuleIndex) ([]*discovery.Module, error) {
	gitClient := git.NewClient(workDir)

	if !gitClient.IsGitRepo() {
		return nil, fmt.Errorf("not a git repository: %s", workDir)
	}

	detector := git.NewChangedModulesDetector(gitClient, moduleIndex, workDir)

	// Determine base ref
	ref := baseRef
	if ref == "" {
		ref = gitClient.GetDefaultBranch()
	}

	return detector.DetectChangedModules(ref)
}
