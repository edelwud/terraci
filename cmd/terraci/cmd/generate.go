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
	applyGenerateCLIFlags(cmd)

	// Discover and filter modules
	allModules, modules, err := discoverAndFilterModules()
	if err != nil {
		return err
	}

	// Build module indexes
	fullModuleIndex := discovery.NewModuleIndex(allModules)
	moduleIndex := discovery.NewModuleIndex(modules)

	// Parse dependencies and build graph
	depGraph := buildDependencyGraph(modules, moduleIndex)

	// Determine target modules
	targetModules, err := determineTargetModules(modules, fullModuleIndex, moduleIndex, depGraph)
	if err != nil {
		return err
	}
	if targetModules == nil {
		return nil // No modules to process
	}

	// Generate and output pipeline
	return generateAndOutputPipeline(targetModules, modules, depGraph)
}

// applyGenerateCLIFlags applies CLI flag overrides to configuration
func applyGenerateCLIFlags(cmd *cobra.Command) {
	if cmd.Flags().Changed("auto-approve") {
		cfg.GitLab.AutoApprove = true
	} else if cmd.Flags().Changed("no-auto-approve") {
		cfg.GitLab.AutoApprove = false
	}

	if planOnly {
		cfg.GitLab.PlanOnly = true
		cfg.GitLab.PlanEnabled = true
	}
}

// discoverAndFilterModules scans for modules and applies filters
func discoverAndFilterModules() (allModules, filteredModules []*discovery.Module, err error) {
	log.WithField("dir", workDir).Info("scanning for terraform modules")

	scanner := discovery.NewScanner(workDir)
	scanner.MinDepth = cfg.Structure.MinDepth
	scanner.MaxDepth = cfg.Structure.MaxDepth

	allModules, err = scanner.Scan()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to scan modules: %w", err)
	}

	log.WithField("count", len(allModules)).Info("discovered modules")

	if len(allModules) == 0 {
		return nil, nil, fmt.Errorf("no modules found in %s", workDir)
	}

	if changedOnly {
		log.WithField("count", len(allModules)).Debug("full module index built")
	}

	modules := applyFilters(allModules)

	if len(modules) != len(allModules) {
		log.WithField("before", len(allModules)).WithField("after", len(modules)).Info("filtered modules")
	}

	if len(modules) == 0 && !changedOnly {
		return nil, nil, fmt.Errorf("no modules remaining after filtering")
	}

	return allModules, modules, nil
}

// buildDependencyGraph parses dependencies and builds the graph
func buildDependencyGraph(modules []*discovery.Module, moduleIndex *discovery.ModuleIndex) *graph.DependencyGraph {
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

	log.Debug("building dependency graph")
	depGraph := graph.BuildFromDependencies(modules, deps)

	cycles := depGraph.DetectCycles()
	if len(cycles) > 0 {
		log.WithField("count", len(cycles)).Warn("circular dependencies detected")
		log.IncreasePadding()
		for _, cycle := range cycles {
			log.WithField("cycle", fmt.Sprintf("%v", cycle)).Warn("cycle found")
		}
		log.DecreasePadding()
	}

	return depGraph
}

// determineTargetModules determines which modules to include in the pipeline
func determineTargetModules(
	modules []*discovery.Module,
	fullModuleIndex, moduleIndex *discovery.ModuleIndex,
	depGraph *graph.DependencyGraph,
) ([]*discovery.Module, error) {
	targetModules := modules

	if changedOnly {
		var err error
		targetModules, err = detectChangedTargetModules(fullModuleIndex, moduleIndex, depGraph)
		if err != nil {
			return nil, err
		}
	}

	if len(targetModules) == 0 {
		log.Info("no modules to process")
		return nil, nil
	}

	return targetModules, nil
}

// generateAndOutputPipeline generates the pipeline and writes output
func generateAndOutputPipeline(
	targetModules, allFilteredModules []*discovery.Module,
	depGraph *graph.DependencyGraph,
) error {
	log.WithField("modules", len(targetModules)).Info("generating pipeline")
	generator := gitlab.NewGenerator(cfg, depGraph, allFilteredModules)

	if dryRun {
		return runDryRun(generator, targetModules)
	}

	pipeline, err := generator.Generate(targetModules)
	if err != nil {
		return fmt.Errorf("failed to generate pipeline: %w", err)
	}

	return writePipelineOutput(pipeline)
}

// runDryRun executes a dry run and outputs results
func runDryRun(generator *gitlab.Generator, targetModules []*discovery.Module) error {
	result, err := generator.DryRun(targetModules)
	if err != nil {
		return fmt.Errorf("dry run failed: %w", err)
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

// writePipelineOutput writes the pipeline to file or stdout
func writePipelineOutput(pipeline *gitlab.Pipeline) error {
	yamlContent, err := pipeline.ToYAML()
	if err != nil {
		return fmt.Errorf("failed to serialize pipeline: %w", err)
	}

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
	// Combine config excludes/includes with command line flags
	allExcludes := append([]string{}, cfg.Exclude...)
	allExcludes = append(allExcludes, excludes...)
	allIncludes := append([]string{}, cfg.Include...)
	allIncludes = append(allIncludes, includes...)

	return filter.Apply(modules, filter.Options{
		Excludes:     allExcludes,
		Includes:     allIncludes,
		Services:     services,
		Environments: environments,
		Regions:      regions,
	})
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
