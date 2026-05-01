package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/workflow"
)

func newGenerateCmd(app *App) *cobra.Command {
	var (
		outputFile  string
		changedOnly bool
		baseRef     string
		dryRun      bool
		planOnly    bool
	)
	ff := &filter.Flags{}

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
			applyProviderFlags(app, planOnly, cmd)

			result, err := workflow.Run(cmd.Context(), workflowOptions(app, ff))
			if err != nil {
				return err
			}

			if len(result.Filtered.Modules) == 0 && !changedOnly {
				return errors.New("no modules remaining after filtering")
			}

			logExtractionWarnings(result.Warnings)
			logLibraryModuleUsage(result.Graph, app.WorkDir)
			logCycles(result.Graph)

			targets, err := resolveGenerateTargets(cmd.Context(), app, result, changedOnly, baseRef, ff)
			if err != nil {
				return err
			}

			if len(targets) == 0 {
				log.Info("no modules to process")
				return nil
			}

			log.WithField("modules", len(targets)).Info("generating pipeline")
			generator, genErr := newPipelineGenerator(app, result.Graph, result.Filtered.Modules, targets, planOnly)
			if genErr != nil {
				return genErr
			}

			if dryRun {
				return runDryRun(generator)
			}

			p, err := generator.Generate()
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
	registerFilterFlags(cmd, ff)

	return cmd
}

func resolveGenerateTargets(
	ctx context.Context,
	app *App,
	result *workflow.Result,
	changedOnly bool,
	baseRef string,
	ff *filter.Flags,
) ([]*discovery.Module, error) {
	return workflow.ResolveTargets(ctx, app.WorkDir, app.Config, result, workflow.TargetSelectionOptions{
		ChangedOnly: changedOnly,
		BaseRef:     baseRef,
		Filters:     ff,
		ChangeDetectorResolver: func() (workflow.ChangeDetector, error) {
			return app.Plugins.ResolveChangeDetector()
		},
	})
}

// applyProviderFlags applies CLI override flags (--plan-only, --auto-approve) to the provider config.
func applyProviderFlags(app *App, planOnly bool, cmd *cobra.Command) {
	if !planOnly && !cmd.Flags().Changed("auto-approve") && !cmd.Flags().Changed("no-auto-approve") {
		return
	}
	resolved, err := app.Plugins.ResolveCIProvider()
	if err != nil {
		log.WithError(err).Debug("cannot apply CLI flags: provider not resolved")
		return
	}

	fo, ok := resolved.Plugin().(plugin.FlagOverridable)
	if !ok {
		log.Debug("CI provider does not support flag overrides")
		return
	}

	if planOnly {
		fo.SetPlanOnly(true)
	}
	if cmd.Flags().Changed("auto-approve") {
		fo.SetAutoApprove(true)
	} else if cmd.Flags().Changed("no-auto-approve") {
		fo.SetAutoApprove(false)
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

func newPipelineGenerator(app *App, depGraph *graph.DependencyGraph, modules, targets []*discovery.Module, planOnly bool) (pipeline.Generator, error) {
	provider, err := app.Plugins.ResolveCIProvider()
	if err != nil {
		return nil, fmt.Errorf("resolve CI provider: %w", err)
	}
	appCtx := app.PluginContext()
	contributions := app.Plugins.CollectContributions(appCtx)

	exec := app.Config.Execution
	ir, err := pipeline.Build(pipeline.BuildOptions{
		DepGraph:      depGraph,
		TargetModules: targets,
		AllModules:    modules,
		ModuleIndex:   discovery.NewModuleIndex(modules),
		Contributions: contributions,
		PlanEnabled:   exec.PlanEnabled,
		PlanOnly:      planOnly,
		Script: pipeline.ScriptConfig{
			InitEnabled: exec.InitEnabled,
			PlanEnabled: exec.PlanEnabled,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("build pipeline IR: %w", err)
	}
	return provider.NewGenerator(appCtx, ir), nil
}

func runDryRun(gen pipeline.Generator) error {
	result, err := gen.DryRun()
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
func makeRelative(path, base string) string {
	if absBase, err := filepath.Abs(base); err == nil {
		if rel, err := filepath.Rel(absBase, path); err == nil {
			return rel
		}
	}
	return path
}
