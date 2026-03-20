package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/graph"
	"github.com/edelwud/terraci/internal/parser"
	"github.com/edelwud/terraci/pkg/log"
)

var (
	graphFormat    string
	graphOutput    string
	showStats      bool
	moduleID       string
	showDependents bool
)

var graphCmd = &cobra.Command{
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
	RunE: runGraph,
}

func init() {
	rootCmd.AddCommand(graphCmd)

	graphCmd.Flags().StringVarP(&graphFormat, "format", "F", "dot", "output format: dot, plantuml, list, levels")
	graphCmd.Flags().StringVarP(&graphOutput, "output", "o", "", "output file (default: stdout)")
	graphCmd.Flags().BoolVar(&showStats, "stats", false, "show graph statistics")
	graphCmd.Flags().StringVarP(&moduleID, "module", "m", "", "filter to specific module")
	graphCmd.Flags().BoolVar(&showDependents, "dependents", false, "show dependents instead of dependencies (with --module)")

	registerFilterFlags(graphCmd)
}

func runGraph(_ *cobra.Command, _ []string) error {
	depGraph, err := buildGraphFromModules()
	if err != nil {
		return err
	}

	if moduleID != "" {
		depGraph, err = scopeToModule(depGraph)
		if err != nil {
			return err
		}
	}

	if showStats {
		return printStats(depGraph)
	}

	return renderGraph(depGraph)
}

// --- Graph construction ---

func buildGraphFromModules() (*graph.DependencyGraph, error) {
	log.WithField("dir", workDir).Debug("scanning for modules")

	scanner := discovery.NewScanner(workDir, cfg.Structure.MinDepth, cfg.Structure.MaxDepth, cfg.Structure.Segments)

	allModules, err := scanner.Scan()
	if err != nil {
		return nil, fmt.Errorf("scan modules: %w", err)
	}
	if len(allModules) == 0 {
		return nil, fmt.Errorf("no modules found in %s", workDir)
	}

	modules := applyFilters(allModules)
	log.WithField("count", len(modules)).Debug("modules after filtering")

	moduleIndex := discovery.NewModuleIndex(modules)

	hclParser := parser.NewParser()
	if len(cfg.Structure.Segments) > 0 {
		hclParser.Segments = cfg.Structure.Segments
	}

	deps, _ := parser.NewDependencyExtractor(hclParser, moduleIndex).ExtractAllDependencies()
	return graph.BuildFromDependencies(modules, deps), nil
}

func scopeToModule(g *graph.DependencyGraph) (*graph.DependencyGraph, error) {
	if g.GetNode(moduleID) == nil {
		return nil, fmt.Errorf("module not found: %s", moduleID)
	}

	var ids []string
	if showDependents {
		ids = append([]string{moduleID}, g.GetAllDependents(moduleID)...)
	} else {
		ids = append([]string{moduleID}, g.GetAllDependencies(moduleID)...)
	}

	return g.Subgraph(ids), nil
}

// --- Output rendering ---

func renderGraph(g *graph.DependencyGraph) error {
	output, err := formatGraph(g, graphFormat)
	if err != nil {
		return err
	}

	if graphOutput != "" {
		if err := os.WriteFile(graphOutput, []byte(output), 0o600); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		log.WithField("file", graphOutput).Info("graph written")
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
	for _, id := range sorted {
		deps := g.GetDependencies(id)
		if len(deps) == 0 {
			fmt.Fprintf(&sb, "%s\n", id)
		} else {
			fmt.Fprintf(&sb, "%s -> %s\n", id, strings.Join(deps, ", "))
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
		fmt.Fprintf(&sb, "Level %d:\n", i)
		for _, id := range level {
			fmt.Fprintf(&sb, "  - %s\n", id)
		}
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

// --- Stats ---

func printStats(g *graph.DependencyGraph) error {
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
	log.WithField("depth", stats.MaxDepth).Info("max depth")
	log.WithField("depth", fmt.Sprintf("%.2f", stats.AverageDepth)).Info("average depth")

	if stats.HasCycles {
		log.WithField("count", stats.CycleCount).Warn("cycles detected")
		log.IncreasePadding()
		for i, cycle := range g.DetectCycles() {
			log.WithField("cycle", i+1).WithField("path", strings.Join(cycle, " -> ")).Warn("cycle")
		}
		log.DecreasePadding()
	} else {
		log.Info("no cycles")
	}

	log.DecreasePadding()
	return nil
}
