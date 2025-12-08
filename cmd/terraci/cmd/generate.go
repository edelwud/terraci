package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/filter"
	"github.com/edelwud/terraci/internal/git"
	"github.com/edelwud/terraci/internal/graph"
	"github.com/edelwud/terraci/internal/parser"
	"github.com/edelwud/terraci/internal/pipeline/gitlab"
	"github.com/spf13/cobra"
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
  terraci generate --exclude "*/test/*" --exclude "platform/*/eu-north-1/*"

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

	allModules, err := scanner.Scan()
	if err != nil {
		return fmt.Errorf("failed to scan modules: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Found %d modules\n", len(allModules))
	}

	if len(allModules) == 0 {
		return fmt.Errorf("no modules found in %s", workDir)
	}

	// 2. Build full module index (before filtering) for change detection
	fullModuleIndex := discovery.NewModuleIndex(allModules)

	if verbose && changedOnly {
		fmt.Fprintf(os.Stderr, "Full module index contains %d modules\n", len(allModules))
	}

	// 3. Apply filters
	modules := applyFilters(allModules)

	if verbose {
		fmt.Fprintf(os.Stderr, "After filtering: %d modules\n", len(modules))
	}

	if len(modules) == 0 && !changedOnly {
		return fmt.Errorf("no modules remaining after filtering")
	}

	// 4. Build filtered module index
	moduleIndex := discovery.NewModuleIndex(modules)

	// 5. Parse dependencies
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

	// 6. Build dependency graph
	depGraph := graph.BuildFromDependencies(modules, deps)

	// Check for cycles
	cycles := depGraph.DetectCycles()
	if len(cycles) > 0 {
		fmt.Fprintf(os.Stderr, "WARNING: Circular dependencies detected:\n")
		for _, cycle := range cycles {
			fmt.Fprintf(os.Stderr, "  %v\n", cycle)
		}
	}

	// 7. Determine target modules
	targetModules := modules

	if changedOnly {
		// Use full module index to detect changes (before filtering)
		changedModules, changedFiles, err := getChangedModulesVerbose(fullModuleIndex)
		if err != nil {
			return fmt.Errorf("failed to detect changed modules: %w", err)
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "Git changed files: %d\n", len(changedFiles))
			checkedDirs := make(map[string]bool)
			for _, f := range changedFiles {
				// Check if the directory exists in the module index
				dir := filepath.Dir(f)
				if !checkedDirs[dir] {
					checkedDirs[dir] = true
					if m := fullModuleIndex.ByPath(dir); m != nil {
						fmt.Fprintf(os.Stderr, "  %s -> module: %s\n", dir, m.ID())
					} else {
						// Check if directory physically exists
						absDir := filepath.Join(workDir, dir)
						if info, err := os.Stat(absDir); err == nil && info.IsDir() {
							// Check for .tf files
							entries, _ := os.ReadDir(absDir)
							var tfCount int
							for _, e := range entries {
								if !e.IsDir() && filepath.Ext(e.Name()) == ".tf" {
									tfCount++
								}
							}
							fmt.Fprintf(os.Stderr, "  %s -> NOT INDEXED (dir exists with %d .tf files)\n", dir, tfCount)
						} else {
							fmt.Fprintf(os.Stderr, "  %s -> DELETED (dir not on disk)\n", dir)
						}
					}
				}
			}
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

		// Also include the changed modules themselves if they pass filters
		affectedSet := make(map[string]bool)
		for _, id := range affectedIDs {
			affectedSet[id] = true
		}
		for _, id := range changedIDs {
			affectedSet[id] = true
		}

		targetModules = make([]*discovery.Module, 0, len(affectedSet))
		for id := range affectedSet {
			// First try filtered index
			if m := moduleIndex.ByID(id); m != nil {
				targetModules = append(targetModules, m)
			} else if m := fullModuleIndex.ByID(id); m != nil {
				// If not in filtered index, check if it passes filters
				filtered := applyFilters([]*discovery.Module{m})
				if len(filtered) > 0 {
					targetModules = append(targetModules, m)
				} else if verbose {
					fmt.Fprintf(os.Stderr, "  (filtered out: %s)\n", m.ID())
				}
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
# https://github.com/edelwud/terraci

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
	modules, _, err := getChangedModulesVerbose(moduleIndex)
	return modules, err
}

func getChangedModulesVerbose(moduleIndex *discovery.ModuleIndex) ([]*discovery.Module, []string, error) {
	gitClient := git.NewClient(workDir)

	if !gitClient.IsGitRepo() {
		return nil, nil, fmt.Errorf("not a git repository: %s", workDir)
	}

	detector := git.NewChangedModulesDetector(gitClient, moduleIndex, workDir)

	// Determine base ref
	ref := baseRef
	if ref == "" {
		ref = gitClient.GetDefaultBranch()
	}

	return detector.DetectChangedModulesVerbose(ref)
}
