package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/workflow"
)

func newGraphCmd(app *App) *cobra.Command {
	var (
		graphFormat    string
		graphOutput    string
		showStats      bool
		moduleID       string
		showDependents bool
	)
	ff := &filterFlags{}

	cmd := &cobra.Command{
		Use:   "graph",
		Short: "Display dependency graph",
		Long: `Display the module dependency graph in various formats.

Formats:
  - dot:      GraphViz DOT format
  - plantuml: PlantUML format
  - list:     Simple text list
  - levels:   Execution levels (parallel groups)

Examples:
  terraci graph --format dot -o deps.dot
  terraci graph --format dot | dot -Tpng -o deps.png
  terraci graph --format plantuml -o deps.puml
  terraci graph --format levels
  terraci graph --stats
  terraci graph --module platform/stage/eu-central-1/vpc --dependents`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := workflow.Run(cmd.Context(), ff.workflowOptions(app))
			if err != nil {
				return err
			}

			log.WithField("count", len(result.FilteredModules)).Debug("modules after filtering")
			depGraph := result.Graph

			if moduleID != "" {
				depGraph, err = depGraph.ScopeToModule(moduleID, showDependents)
				if err != nil {
					return err
				}
			}

			if showStats {
				return printStats(depGraph, moduleID)
			}

			return renderGraph(depGraph, graphFormat, graphOutput)
		},
	}

	cmd.Flags().StringVarP(&graphFormat, "format", "F", "dot", "output format: dot, plantuml, list, levels")
	cmd.Flags().StringVarP(&graphOutput, "output", "o", "", "output file (default: stdout)")
	cmd.Flags().BoolVar(&showStats, "stats", false, "show graph statistics")
	cmd.Flags().StringVarP(&moduleID, "module", "m", "", "filter to specific module")
	cmd.Flags().BoolVar(&showDependents, "dependents", false, "show dependents instead of dependencies (with --module)")
	ff.register(cmd)

	return cmd
}

func renderGraph(g *graph.DependencyGraph, format, outputFile string) error {
	output, err := formatGraph(g, format)
	if err != nil {
		return err
	}

	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(output), 0o600); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		log.WithField("file", outputFile).Info("graph written")
		return nil
	}

	fmt.Print(output)
	return nil
}

func formatGraph(g *graph.DependencyGraph, format string) (string, error) {
	switch format {
	case "dot":
		return g.ToDOT(), nil
	case "plantuml":
		return g.ToPlantUML(), nil
	case "list":
		return formatList(g)
	case "levels":
		return formatLevels(g)
	default:
		return "", fmt.Errorf("unknown format: %s", format)
	}
}

func formatList(g *graph.DependencyGraph) (string, error) {
	sorted, err := g.TopologicalSort()
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	currentGroup := ""
	for _, id := range sorted {
		parts := strings.Split(id, "/")
		group := ""
		if len(parts) >= 2 {
			group = parts[0] + "/" + parts[1]
		}

		if group != currentGroup {
			if currentGroup != "" {
				sb.WriteString("\n")
			}
			fmt.Fprintf(&sb, "[%s]\n", group)
			currentGroup = group
		}

		shortName := id
		if len(parts) > 2 {
			shortName = strings.Join(parts[2:], "/")
		}

		deps := g.GetDependencies(id)
		if len(deps) == 0 {
			fmt.Fprintf(&sb, "  %s\n", shortName)
		} else {
			shortDeps := make([]string, len(deps))
			for i, dep := range deps {
				depParts := strings.Split(dep, "/")
				if len(depParts) > 2 {
					shortDeps[i] = strings.Join(depParts[2:], "/")
				} else {
					shortDeps[i] = dep
				}
			}
			fmt.Fprintf(&sb, "  %s → %s\n", shortName, strings.Join(shortDeps, ", "))
		}
	}

	return sb.String(), nil
}

func formatLevels(g *graph.DependencyGraph) (string, error) {
	levels, err := g.ExecutionLevels()
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for i, level := range levels {
		fmt.Fprintf(&sb, "Level %d (%d modules):\n", i, len(level))
		for _, id := range level {
			deps := g.GetDependencies(id)
			if len(deps) == 0 {
				fmt.Fprintf(&sb, "  %s\n", id)
			} else {
				depNames := make([]string, len(deps))
				for j, dep := range deps {
					parts := strings.Split(dep, "/")
					depNames[j] = parts[len(parts)-1]
				}
				fmt.Fprintf(&sb, "  %s  (← %s)\n", id, strings.Join(depNames, ", "))
			}
		}
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

func printStats(g *graph.DependencyGraph, moduleID string) error {
	stats := g.GetStats()

	if moduleID != "" {
		log.WithField("scope", moduleID).Info("dependency graph statistics")
	} else {
		log.Info("dependency graph statistics")
	}

	log.IncreasePadding()

	log.WithField("count", stats.TotalModules).Info("total modules")
	log.WithField("count", stats.TotalEdges).Info("total edges")
	log.WithField("count", stats.RootModules).Info("root modules (no dependencies)")
	log.WithField("count", stats.LeafModules).Info("leaf modules (no dependents)")
	log.WithField("depth", stats.MaxDepth).Info("max depth (execution levels)")
	log.WithField("depth", fmt.Sprintf("%.1f", stats.AverageDepth)).Info("average depth")

	if len(stats.LevelCounts) > 0 {
		levelStrs := make([]string, len(stats.LevelCounts))
		for i, c := range stats.LevelCounts {
			levelStrs[i] = fmt.Sprintf("L%d:%d", i, c)
		}
		log.WithField("distribution", strings.Join(levelStrs, " ")).Info("modules per level")
	}

	if len(stats.TopDependedOn) > 0 {
		log.Info("most depended-on modules (bottlenecks)")
		log.IncreasePadding()
		for _, m := range stats.TopDependedOn {
			log.WithField("dependents", m.Count).Info(m.ID)
		}
		log.DecreasePadding()
	}

	if len(stats.TopDependencies) > 0 {
		log.Info("modules with most dependencies")
		log.IncreasePadding()
		for _, m := range stats.TopDependencies {
			log.WithField("dependencies", m.Count).Info(m.ID)
		}
		log.DecreasePadding()
	}

	if stats.HasCycles {
		log.WithField("count", stats.CycleCount).Warn("cycles detected")
		log.IncreasePadding()
		for i, cycle := range g.DetectCycles() {
			log.WithField("cycle", i+1).WithField("path", strings.Join(cycle, " → ")).Warn("cycle")
		}
		log.DecreasePadding()
	} else {
		log.Info("no cycles ✓")
	}

	log.DecreasePadding()
	return nil
}
