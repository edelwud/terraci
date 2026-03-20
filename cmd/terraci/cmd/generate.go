package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/git"
	"github.com/edelwud/terraci/internal/graph"
	"github.com/edelwud/terraci/internal/parser"
	"github.com/edelwud/terraci/internal/pipeline"
	pipelinegithub "github.com/edelwud/terraci/internal/pipeline/github"
	"github.com/edelwud/terraci/internal/pipeline/gitlab"
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/log"
)

var (
	outputFile  string
	changedOnly bool
	baseRef     string
	dryRun      bool
	planOnly    bool
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate CI pipeline",
	Long: `Generate a CI pipeline YAML file based on the Terraform
module structure and dependencies.

Examples:
  terraci generate -o .gitlab-ci.yml
  terraci generate --changed-only --base-ref main
  terraci generate --exclude "*/test/*"
  terraci generate --filter environment=stage --filter environment=prod
  terraci generate --dry-run
  terraci generate --auto-approve
  terraci generate --plan-only`,
	RunE: runGenerate,
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file (default: stdout)")
	generateCmd.Flags().BoolVar(&changedOnly, "changed-only", false, "only include changed modules and their dependents")
	generateCmd.Flags().StringVar(&baseRef, "base-ref", "", "base git ref for change detection (default: auto-detect)")
	generateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be generated without creating output")
	generateCmd.Flags().BoolVar(&planOnly, "plan-only", false, "generate only plan jobs (no apply jobs)")
	generateCmd.Flags().Bool("auto-approve", false, "auto-approve apply jobs (skip manual trigger)")
	generateCmd.Flags().Bool("no-auto-approve", false, "require manual trigger for apply jobs")
	registerFilterFlags(generateCmd)
}

// --- Main flow ---

func runGenerate(cmd *cobra.Command, _ []string) error {
	applyGenerateCLIFlags(cmd)

	allModules, modules, err := discoverAndFilterModules()
	if err != nil {
		return err
	}

	fullIndex := discovery.NewModuleIndex(allModules)
	filteredIndex := discovery.NewModuleIndex(modules)

	depGraph := buildDependencyGraph(modules, filteredIndex)

	targets, err := determineTargetModules(modules, fullIndex, filteredIndex, depGraph)
	if err != nil {
		return err
	}
	if targets == nil {
		return nil
	}

	return generateAndOutputPipeline(targets, modules, depGraph)
}

// --- CLI flag application ---

func applyGenerateCLIFlags(cmd *cobra.Command) {
	provider := config.ResolveProvider(cfg)

	if planOnly {
		applyPlanOnly(provider)
	}

	if cmd.Flags().Changed("auto-approve") {
		setAutoApprove(provider, true)
	} else if cmd.Flags().Changed("no-auto-approve") {
		setAutoApprove(provider, false)
	}
}

func applyPlanOnly(provider string) {
	switch provider {
	case config.ProviderGitHub:
		if cfg.GitHub != nil {
			cfg.GitHub.PlanOnly = true
			cfg.GitHub.PlanEnabled = true
		}
	default:
		if cfg.GitLab != nil {
			cfg.GitLab.PlanOnly = true
			cfg.GitLab.PlanEnabled = true
		}
	}
}

func setAutoApprove(provider string, approve bool) {
	switch provider {
	case config.ProviderGitHub:
		if cfg.GitHub != nil {
			cfg.GitHub.AutoApprove = approve
		}
	default:
		if cfg.GitLab != nil {
			cfg.GitLab.AutoApprove = approve
		}
	}
}

// --- Module discovery and filtering ---

func discoverAndFilterModules() (allModules, filteredModules []*discovery.Module, err error) {
	log.WithField("dir", workDir).Info("scanning for terraform modules")

	scanner := discovery.NewScanner(workDir, cfg.Structure.MinDepth, cfg.Structure.MaxDepth, cfg.Structure.Segments)

	allModules, err = scanner.Scan()
	if err != nil {
		return nil, nil, fmt.Errorf("scan modules: %w", err)
	}

	log.WithField("count", len(allModules)).Info("discovered modules")

	if len(allModules) == 0 {
		return nil, nil, fmt.Errorf("no modules found in %s", workDir)
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

// --- Dependency graph ---

func buildDependencyGraph(modules []*discovery.Module, moduleIndex *discovery.ModuleIndex) *graph.DependencyGraph {
	log.Info("parsing module dependencies")

	hclParser := parser.NewParser()
	if len(cfg.Structure.Segments) > 0 {
		hclParser.Segments = cfg.Structure.Segments
	}

	deps, errs := parser.NewDependencyExtractor(hclParser, moduleIndex).ExtractAllDependencies()
	logExtractionWarnings(errs)

	depGraph := graph.BuildFromDependencies(modules, deps)
	logLibraryModuleUsage(depGraph)
	logCycles(depGraph)

	return depGraph
}

func logExtractionWarnings(errs []error) {
	if len(errs) == 0 {
		return
	}
	log.WithField("count", len(errs)).Warn("warnings during dependency extraction")
	log.IncreasePadding()
	for _, e := range errs {
		log.WithField("warning", e.Error()).Debug("extraction warning")
	}
	log.DecreasePadding()
}

func logLibraryModuleUsage(depGraph *graph.DependencyGraph) {
	paths := depGraph.GetAllLibraryPaths()
	if len(paths) == 0 {
		return
	}

	log.WithField("count", len(paths)).Info("library modules detected")
	log.IncreasePadding()
	for _, libPath := range paths {
		users := depGraph.GetModulesUsingLibrary(libPath)
		relPath := makeRelative(libPath, workDir)
		log.WithField("path", relPath).WithField("used_by", len(users)).Debug("library module")
	}
	log.DecreasePadding()
}

func logCycles(depGraph *graph.DependencyGraph) {
	cycles := depGraph.DetectCycles()
	if len(cycles) == 0 {
		return
	}
	log.WithField("count", len(cycles)).Warn("circular dependencies detected")
	log.IncreasePadding()
	for _, cycle := range cycles {
		log.WithField("cycle", fmt.Sprintf("%v", cycle)).Warn("cycle found")
	}
	log.DecreasePadding()
}

// --- Target module determination ---

func determineTargetModules(
	modules []*discovery.Module,
	fullIndex, filteredIndex *discovery.ModuleIndex,
	depGraph *graph.DependencyGraph,
) ([]*discovery.Module, error) {
	targets := modules

	if changedOnly {
		var err error
		targets, err = detectChangedTargetModules(fullIndex, filteredIndex, depGraph)
		if err != nil {
			return nil, err
		}
	}

	if len(targets) == 0 {
		log.Info("no modules to process")
		return nil, nil
	}

	return targets, nil
}

// --- Pipeline generation ---

func generateAndOutputPipeline(
	targets, allFiltered []*discovery.Module,
	depGraph *graph.DependencyGraph,
) error {
	log.WithField("modules", len(targets)).Info("generating pipeline")

	generator := newPipelineGenerator(depGraph, allFiltered)

	if dryRun {
		return runDryRun(generator, targets)
	}

	p, err := generator.Generate(targets)
	if err != nil {
		return fmt.Errorf("generate pipeline: %w", err)
	}
	return writePipelineOutput(p)
}

func newPipelineGenerator(depGraph *graph.DependencyGraph, modules []*discovery.Module) pipeline.Generator {
	switch config.ResolveProvider(cfg) {
	case config.ProviderGitHub:
		return pipelinegithub.NewGenerator(cfg, depGraph, modules)
	default:
		return gitlab.NewGenerator(cfg, depGraph, modules)
	}
}

func runDryRun(gen pipeline.Generator, targets []*discovery.Module) error {
	result, err := gen.DryRun(targets)
	if err != nil {
		return fmt.Errorf("dry run: %w", err)
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

func writePipelineOutput(p pipeline.GeneratedPipeline) error {
	yaml, err := p.ToYAML()
	if err != nil {
		return fmt.Errorf("serialize pipeline: %w", err)
	}

	content := append([]byte("# Generated by terraci\n# DO NOT EDIT - this file is auto-generated\n# https://github.com/edelwud/terraci\n\n"), yaml...)

	if outputFile != "" {
		if err := os.WriteFile(outputFile, content, 0o600); err != nil {
			return fmt.Errorf("write output file: %w", err)
		}
		log.WithField("file", outputFile).Info("pipeline written")
	} else {
		fmt.Print(string(content))
	}

	return nil
}

// --- Changed module detection ---

func detectChangedTargetModules(
	fullIndex, filteredIndex *discovery.ModuleIndex,
	depGraph *graph.DependencyGraph,
) ([]*discovery.Module, error) {
	log.Info("detecting changed modules")

	changedModules, changedFiles, err := getChangedModulesVerbose(fullIndex)
	if err != nil {
		return nil, fmt.Errorf("detect changed modules: %w", err)
	}

	log.WithField("files", len(changedFiles)).Debug("git changes detected")
	log.WithField("count", len(changedModules)).Info("changed modules detected")

	changedIDs := moduleIDs(changedModules)
	libraryPaths := detectChangedLibraries()

	affectedIDs := computeAffectedIDs(depGraph, changedIDs, libraryPaths)

	targets := resolveAffectedModules(affectedIDs, changedIDs, fullIndex, filteredIndex)
	log.WithField("count", len(targets)).Info("affected modules (including dependents)")

	return targets, nil
}

func detectChangedLibraries() []string {
	if cfg.LibraryModules == nil || len(cfg.LibraryModules.Paths) == 0 {
		return nil
	}

	log.Debug("checking for changed library modules")
	gitClient := git.NewClient(workDir)
	detector := git.NewChangedModulesDetector(gitClient, discovery.NewModuleIndex(nil), workDir)

	ref := baseRef
	if ref == "" {
		ref = gitClient.GetDefaultBranch()
	}

	paths, err := detector.DetectChangedLibraryModules(ref, cfg.LibraryModules.Paths)
	if err != nil {
		log.WithError(err).Warn("failed to detect changed library modules")
		return nil
	}

	if len(paths) > 0 {
		log.WithField("count", len(paths)).Info("changed library modules")
	}
	return paths
}

func computeAffectedIDs(depGraph *graph.DependencyGraph, changedIDs, libraryPaths []string) []string {
	if len(libraryPaths) > 0 {
		return depGraph.GetAffectedModulesWithLibraries(changedIDs, libraryPaths)
	}
	return depGraph.GetAffectedModules(changedIDs)
}

func resolveAffectedModules(
	affectedIDs, changedIDs []string,
	fullIndex, filteredIndex *discovery.ModuleIndex,
) []*discovery.Module {
	idSet := make(map[string]bool, len(affectedIDs)+len(changedIDs))
	for _, id := range affectedIDs {
		idSet[id] = true
	}
	for _, id := range changedIDs {
		idSet[id] = true
	}

	targets := make([]*discovery.Module, 0, len(idSet))
	for id := range idSet {
		if m := filteredIndex.ByID(id); m != nil {
			targets = append(targets, m)
		} else if m := fullIndex.ByID(id); m != nil {
			if filtered := applyFilters([]*discovery.Module{m}); len(filtered) > 0 {
				targets = append(targets, m)
			} else {
				log.WithField("module", m.ID()).Debug("filtered out")
			}
		}
	}

	return targets
}

// --- Helpers ---

func moduleIDs(modules []*discovery.Module) []string {
	ids := make([]string, len(modules))
	for i, m := range modules {
		ids[i] = m.ID()
	}
	return ids
}

func makeRelative(path, base string) string {
	if absBase, err := filepath.Abs(base); err == nil {
		if rel, err := filepath.Rel(absBase, path); err == nil {
			return rel
		}
	}
	return path
}

func getChangedModulesVerbose(moduleIndex *discovery.ModuleIndex) ([]*discovery.Module, []string, error) {
	gitClient := git.NewClient(workDir)
	if !gitClient.IsGitRepo() {
		return nil, nil, fmt.Errorf("not a git repository: %s", workDir)
	}

	ref := baseRef
	if ref == "" {
		ref = gitClient.GetDefaultBranch()
	}

	return git.NewChangedModulesDetector(gitClient, moduleIndex, workDir).DetectChangedModulesVerbose(ref)
}
