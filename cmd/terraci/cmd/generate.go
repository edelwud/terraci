package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/filter"
	"github.com/edelwud/terraci/internal/git"
	"github.com/edelwud/terraci/internal/graph"
	"github.com/edelwud/terraci/internal/parser"
	"github.com/edelwud/terraci/internal/pipeline/gitlab"
	"github.com/edelwud/terraci/pkg/log"
)

var (
	// Generate command flags
	outputFile   string
	changedOnly  bool
	baseRef      string
	excludes     []string
	includes     []string
	dryRun       bool
	planOnly     bool
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
  terraci generate --dry-run

  # Generate with auto-approve (skip manual trigger for apply jobs)
  terraci generate --auto-approve

  # Generate only plan jobs (no apply jobs)
  terraci generate --plan-only`,
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
	generateCmd.Flags().BoolVar(&planOnly, "plan-only", false, "generate only plan jobs (no apply jobs)")

	// Auto-approve flag with explicit true/false handling
	generateCmd.Flags().Bool("auto-approve", false, "auto-approve apply jobs (skip manual trigger)")
	generateCmd.Flags().Bool("no-auto-approve", false, "require manual trigger for apply jobs")
}

func runGenerate(cmd *cobra.Command, _ []string) error {
	// Handle auto-approve flags (CLI overrides config)
	if cmd.Flags().Changed("auto-approve") {
		cfg.GitLab.AutoApprove = true
	} else if cmd.Flags().Changed("no-auto-approve") {
		cfg.GitLab.AutoApprove = false
	}

	// Handle plan-only flag (CLI overrides config)
	if planOnly {
		cfg.GitLab.PlanOnly = true
		// PlanOnly implies PlanEnabled
		cfg.GitLab.PlanEnabled = true
	}

	// 1. Discover modules
	log.WithField("dir", workDir).Info("scanning for terraform modules")

	scanner := discovery.NewScanner(workDir)
	scanner.MinDepth = cfg.Structure.MinDepth
	scanner.MaxDepth = cfg.Structure.MaxDepth

	allModules, err := scanner.Scan()
	if err != nil {
		return fmt.Errorf("failed to scan modules: %w", err)
	}

	log.WithField("count", len(allModules)).Info("discovered modules")

	if len(allModules) == 0 {
		return fmt.Errorf("no modules found in %s", workDir)
	}

	// 2. Build full module index (before filtering) for change detection
	fullModuleIndex := discovery.NewModuleIndex(allModules)

	if changedOnly {
		log.WithField("count", len(allModules)).Debug("full module index built")
	}

	// 3. Apply filters
	modules := applyFilters(allModules)

	if len(modules) != len(allModules) {
		log.WithField("before", len(allModules)).WithField("after", len(modules)).Info("filtered modules")
	}

	if len(modules) == 0 && !changedOnly {
		return fmt.Errorf("no modules remaining after filtering")
	}

	// 4. Build filtered module index
	moduleIndex := discovery.NewModuleIndex(modules)

	// 5. Parse dependencies
	log.Info("parsing module dependencies")

	hclParser := parser.NewParser()
	depExtractor := parser.NewDependencyExtractor(hclParser, moduleIndex)

	deps, errs := depExtractor.ExtractAllDependencies()
	if len(errs) > 0 {
		log.WithField("count", len(errs)).Warn("warnings during dependency extraction")
		log.IncreasePadding()
		for _, e := range errs {
			log.WithField("warning", e.Error()).Debug("extraction warning")
		}
		log.DecreasePadding()
	}

	// 6. Build dependency graph
	log.Debug("building dependency graph")
	depGraph := graph.BuildFromDependencies(modules, deps)

	// Check for cycles
	cycles := depGraph.DetectCycles()
	if len(cycles) > 0 {
		log.WithField("count", len(cycles)).Warn("circular dependencies detected")
		log.IncreasePadding()
		for _, cycle := range cycles {
			log.WithField("cycle", fmt.Sprintf("%v", cycle)).Warn("cycle found")
		}
		log.DecreasePadding()
	}

	// 7. Determine target modules
	targetModules := modules

	if changedOnly {
		targetModules, err = detectChangedTargetModules(fullModuleIndex, moduleIndex, depGraph)
		if err != nil {
			return err
		}
	}

	if len(targetModules) == 0 {
		log.Info("no modules to process")
		return nil
	}

	// 8. Generate pipeline
	log.WithField("modules", len(targetModules)).Info("generating pipeline")
	generator := gitlab.NewGenerator(cfg, depGraph, modules)

	// Handle dry run
	if dryRun {
		result, dryRunErr := generator.DryRun(targetModules)
		if dryRunErr != nil {
			return fmt.Errorf("dry run failed: %w", dryRunErr)
		}

		log.Info("dry run results")
		log.IncreasePadding()
		log.WithField("total", result.TotalModules).Info("modules discovered")
		log.WithField("affected", result.AffectedModules).Info("modules to process")
		log.WithField("stages", result.Stages).Info("pipeline stages")
		log.WithField("jobs", result.Jobs).Info("pipeline jobs")
		log.DecreasePadding()

		log.Info("execution order")
		log.IncreasePadding()
		for i, level := range result.ExecutionOrder {
			log.WithField("level", i).WithField("modules", fmt.Sprintf("%v", level)).Debug("level")
		}
		log.DecreasePadding()
		return nil
	}

	pipeline, err := generator.Generate(targetModules)
	if err != nil {
		return fmt.Errorf("failed to generate pipeline: %w", err)
	}

	// 9. Output
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
		if err := os.WriteFile(outputFile, yamlContent, 0o600); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		log.WithField("file", outputFile).Info("pipeline written")
	} else {
		fmt.Print(string(yamlContent))
	}

	return nil
}

func detectChangedTargetModules(
	fullModuleIndex, moduleIndex *discovery.ModuleIndex,
	depGraph *graph.DependencyGraph,
) ([]*discovery.Module, error) {
	// Use full module index to detect changes (before filtering)
	log.Info("detecting changed modules")

	changedModules, changedFiles, changeErr := getChangedModulesVerbose(fullModuleIndex)
	if changeErr != nil {
		return nil, fmt.Errorf("failed to detect changed modules: %w", changeErr)
	}

	log.WithField("files", len(changedFiles)).Debug("git changes detected")

	// Log changed file details at debug level
	if log.IsDebug() {
		log.IncreasePadding()
		checkedDirs := make(map[string]bool)
		for _, f := range changedFiles {
			dir := filepath.Dir(f)
			if !checkedDirs[dir] {
				checkedDirs[dir] = true
				if m := fullModuleIndex.ByPath(dir); m != nil {
					log.WithField("dir", dir).WithField("module", m.ID()).Debug("mapped to module")
				} else {
					absDir := filepath.Join(workDir, dir)
					if info, statErr := os.Stat(absDir); statErr == nil && info.IsDir() {
						entries, readErr := os.ReadDir(absDir)
						if readErr != nil {
							log.WithField("dir", dir).WithError(readErr).Debug("error reading dir")
							continue
						}
						var tfCount int
						for _, e := range entries {
							if !e.IsDir() && filepath.Ext(e.Name()) == ".tf" {
								tfCount++
							}
						}
						log.WithField("dir", dir).WithField("tf_files", tfCount).Debug("not indexed")
					} else {
						log.WithField("dir", dir).Debug("deleted directory")
					}
				}
			}
		}
		log.DecreasePadding()
	}

	log.WithField("count", len(changedModules)).Info("changed modules detected")
	if log.IsDebug() {
		log.IncreasePadding()
		for _, m := range changedModules {
			log.WithField("module", m.ID()).Debug("changed")
		}
		log.DecreasePadding()
	}

	// Get affected modules (changed + dependents)
	changedIDs := make([]string, len(changedModules))
	for i, m := range changedModules {
		changedIDs[i] = m.ID()
	}

	// Detect changed library modules if configured
	var changedLibraryPaths []string
	if cfg.LibraryModules != nil && len(cfg.LibraryModules.Paths) > 0 {
		log.Debug("checking for changed library modules")
		gitClient := git.NewClient(workDir)
		detector := git.NewChangedModulesDetector(gitClient, fullModuleIndex, workDir)

		ref := baseRef
		if ref == "" {
			ref = gitClient.GetDefaultBranch()
		}

		var err error
		changedLibraryPaths, err = detector.DetectChangedLibraryModules(ref, cfg.LibraryModules.Paths)
		if err != nil {
			log.WithError(err).Warn("failed to detect changed library modules")
		} else if len(changedLibraryPaths) > 0 {
			log.WithField("count", len(changedLibraryPaths)).Info("changed library modules")
			log.IncreasePadding()
			for _, p := range changedLibraryPaths {
				log.WithField("path", p).Debug("library changed")
			}
			log.DecreasePadding()
		}
	}

	// Get affected modules including library module dependents
	var affectedIDs []string
	if len(changedLibraryPaths) > 0 {
		affectedIDs = depGraph.GetAffectedModulesWithLibraries(changedIDs, changedLibraryPaths)
	} else {
		affectedIDs = depGraph.GetAffectedModules(changedIDs)
	}

	// Also include the changed modules themselves if they pass filters
	affectedSet := make(map[string]bool)
	for _, id := range affectedIDs {
		affectedSet[id] = true
	}
	for _, id := range changedIDs {
		affectedSet[id] = true
	}

	targetModules := make([]*discovery.Module, 0, len(affectedSet))
	for id := range affectedSet {
		// First try filtered index
		if m := moduleIndex.ByID(id); m != nil {
			targetModules = append(targetModules, m)
		} else if m := fullModuleIndex.ByID(id); m != nil {
			// If not in filtered index, check if it passes filters
			filtered := applyFilters([]*discovery.Module{m})
			if len(filtered) > 0 {
				targetModules = append(targetModules, m)
			} else {
				log.WithField("module", m.ID()).Debug("filtered out")
			}
		}
	}

	log.WithField("count", len(targetModules)).Info("affected modules (including dependents)")

	return targetModules, nil
}

func applyFilters(modules []*discovery.Module) []*discovery.Module {
	// Combine config excludes with command line excludes
	allExcludes := append([]string{}, cfg.Exclude...)
	allExcludes = append(allExcludes, excludes...)
	allIncludes := append([]string{}, cfg.Include...)
	allIncludes = append(allIncludes, includes...)

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
