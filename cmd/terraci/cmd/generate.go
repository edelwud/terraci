package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/git"
	"github.com/edelwud/terraci/internal/graph"
	"github.com/edelwud/terraci/internal/pipeline"
	pipelinegithub "github.com/edelwud/terraci/internal/pipeline/github"
	"github.com/edelwud/terraci/internal/pipeline/gitlab"
	"github.com/edelwud/terraci/internal/workflow"
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/log"
)

func newGenerateCmd(app *App) *cobra.Command {
	var (
		outputFile  string
		changedOnly bool
		baseRef     string
		dryRun      bool
		planOnly    bool
	)
	ff := &filterFlags{}

	cmd := &cobra.Command{
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Apply CLI flags to config
			provider := config.ResolveProvider(app.Config)
			if planOnly {
				applyPlanOnly(app, provider)
			}
			if cmd.Flags().Changed("auto-approve") {
				setAutoApprove(app, provider, true)
			} else if cmd.Flags().Changed("no-auto-approve") {
				setAutoApprove(app, provider, false)
			}

			result, err := workflow.Run(cmd.Context(), ff.workflowOptions(app))
			if err != nil {
				return err
			}

			if len(result.FilteredModules) == 0 && !changedOnly {
				return fmt.Errorf("no modules remaining after filtering")
			}

			logExtractionWarnings(result.Warnings)
			logLibraryModuleUsage(result.Graph, app.WorkDir)
			logCycles(result.Graph)

			targets := result.FilteredModules
			if changedOnly {
				targets, err = detectChangedTargetModules(app, ff, baseRef, result.FullIndex, result.FilteredIndex, result.Graph)
				if err != nil {
					return err
				}
			}

			if len(targets) == 0 {
				log.Info("no modules to process")
				return nil
			}

			log.WithField("modules", len(targets)).Info("generating pipeline")
			generator := newPipelineGenerator(app, result.Graph, result.FilteredModules)

			if dryRun {
				return runDryRun(generator, targets)
			}

			p, err := generator.Generate(targets)
			if err != nil {
				return fmt.Errorf("generate pipeline: %w", err)
			}
			return writePipelineOutput(p, outputFile)
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file (default: stdout)")
	cmd.Flags().BoolVar(&changedOnly, "changed-only", false, "only include changed modules and their dependents")
	cmd.Flags().StringVar(&baseRef, "base-ref", "", "base git ref for change detection (default: auto-detect)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be generated without creating output")
	cmd.Flags().BoolVar(&planOnly, "plan-only", false, "generate only plan jobs (no apply jobs)")
	cmd.Flags().Bool("auto-approve", false, "auto-approve apply jobs (skip manual trigger)")
	cmd.Flags().Bool("no-auto-approve", false, "require manual trigger for apply jobs")
	ff.register(cmd)

	return cmd
}

// --- CLI flag application ---

func applyPlanOnly(app *App, provider string) {
	switch provider {
	case config.ProviderGitHub:
		if app.Config.GitHub != nil {
			app.Config.GitHub.PlanOnly = true
			app.Config.GitHub.PlanEnabled = true
		}
	default:
		if app.Config.GitLab != nil {
			app.Config.GitLab.PlanOnly = true
			app.Config.GitLab.PlanEnabled = true
		}
	}
}

func setAutoApprove(app *App, provider string, approve bool) {
	switch provider {
	case config.ProviderGitHub:
		if app.Config.GitHub != nil {
			app.Config.GitHub.AutoApprove = approve
		}
	default:
		if app.Config.GitLab != nil {
			app.Config.GitLab.AutoApprove = approve
		}
	}
}

// --- Logging helpers ---

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

func logLibraryModuleUsage(depGraph *graph.DependencyGraph, workDir string) {
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

// --- Pipeline generation ---

func newPipelineGenerator(app *App, depGraph *graph.DependencyGraph, modules []*discovery.Module) pipeline.Generator {
	switch config.ResolveProvider(app.Config) {
	case config.ProviderGitHub:
		return pipelinegithub.NewGenerator(app.Config, depGraph, modules)
	default:
		return gitlab.NewGenerator(app.Config, depGraph, modules)
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

func writePipelineOutput(p pipeline.GeneratedPipeline, outputFile string) error {
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
	app *App,
	ff *filterFlags,
	baseRef string,
	fullIndex, filteredIndex *discovery.ModuleIndex,
	depGraph *graph.DependencyGraph,
) ([]*discovery.Module, error) {
	log.Info("detecting changed modules")

	changedModules, changedFiles, err := getChangedModulesVerbose(app.WorkDir, baseRef, fullIndex)
	if err != nil {
		return nil, fmt.Errorf("detect changed modules: %w", err)
	}

	log.WithField("files", len(changedFiles)).Debug("git changes detected")
	log.WithField("count", len(changedModules)).Info("changed modules detected")

	changedIDs := moduleIDs(changedModules)
	libraryPaths := detectChangedLibraries(app, baseRef)

	var affectedIDs []string
	if len(libraryPaths) > 0 {
		affectedIDs = depGraph.GetAffectedModulesWithLibraries(changedIDs, libraryPaths)
	} else {
		affectedIDs = depGraph.GetAffectedModules(changedIDs)
	}

	targets := resolveAffectedModules(app, ff, affectedIDs, changedIDs, fullIndex, filteredIndex)
	log.WithField("count", len(targets)).Info("affected modules (including dependents)")

	return targets, nil
}

func detectChangedLibraries(app *App, baseRef string) []string {
	if app.Config.LibraryModules == nil || len(app.Config.LibraryModules.Paths) == 0 {
		return nil
	}

	log.Debug("checking for changed library modules")
	gitClient := git.NewClient(app.WorkDir)
	detector := git.NewChangedModulesDetector(gitClient, discovery.NewModuleIndex(nil), app.WorkDir)

	ref := baseRef
	if ref == "" {
		ref = gitClient.GetDefaultBranch()
	}

	paths, err := detector.DetectChangedLibraryModules(ref, app.Config.LibraryModules.Paths)
	if err != nil {
		log.WithError(err).Warn("failed to detect changed library modules")
		return nil
	}

	if len(paths) > 0 {
		log.WithField("count", len(paths)).Info("changed library modules")
	}
	return paths
}

func resolveAffectedModules(
	app *App,
	ff *filterFlags,
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
			if filtered := ff.applyFilters(app, []*discovery.Module{m}); len(filtered) > 0 {
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

func getChangedModulesVerbose(workDir, baseRef string, moduleIndex *discovery.ModuleIndex) ([]*discovery.Module, []string, error) {
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
