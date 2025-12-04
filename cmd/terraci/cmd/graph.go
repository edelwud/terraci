package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/terraci/terraci/internal/discovery"
	"github.com/terraci/terraci/internal/filter"
	"github.com/terraci/terraci/internal/graph"
	"github.com/terraci/terraci/internal/parser"
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
  - dot: GraphViz DOT format (can be rendered with: dot -Tpng -o graph.png)
  - list: Simple text list
  - levels: Show execution levels (parallel groups)

Examples:
  # Output DOT format to file
  terraci graph --format dot -o deps.dot

  # Render graph as PNG
  terraci graph --format dot | dot -Tpng -o deps.png

  # Show execution levels
  terraci graph --format levels

  # Show statistics
  terraci graph --stats

  # Show dependencies for a specific module
  terraci graph --module cdp/stage/eu-central-1/vpc

  # Show what depends on a module
  terraci graph --module cdp/stage/eu-central-1/vpc --dependents`,
	RunE: runGraph,
}

func init() {
	rootCmd.AddCommand(graphCmd)

	graphCmd.Flags().StringVarP(&graphFormat, "format", "f", "dot", "output format: dot, list, levels")
	graphCmd.Flags().StringVarP(&graphOutput, "output", "o", "", "output file (default: stdout)")
	graphCmd.Flags().BoolVar(&showStats, "stats", false, "show graph statistics")
	graphCmd.Flags().StringVarP(&moduleID, "module", "m", "", "show dependencies for specific module")
	graphCmd.Flags().BoolVar(&showDependents, "dependents", false, "show modules that depend on --module (reverse)")

	// Reuse filter flags from generate
	graphCmd.Flags().StringArrayVarP(&excludes, "exclude", "x", nil, "glob patterns to exclude modules")
	graphCmd.Flags().StringArrayVarP(&includes, "include", "i", nil, "glob patterns to include modules")
}

func runGraph(cmd *cobra.Command, args []string) error {
	// 1. Discover modules
	scanner := discovery.NewScanner(workDir)
	scanner.MinDepth = cfg.Structure.MinDepth
	scanner.MaxDepth = cfg.Structure.MaxDepth

	modules, err := scanner.Scan()
	if err != nil {
		return fmt.Errorf("failed to scan modules: %w", err)
	}

	if len(modules) == 0 {
		return fmt.Errorf("no modules found in %s", workDir)
	}

	// 2. Apply filters
	allExcludes := append(cfg.Exclude, excludes...)
	allIncludes := append(cfg.Include, includes...)
	globFilter := filter.NewGlobFilter(allExcludes, allIncludes)
	modules = globFilter.FilterModules(modules)

	// 3. Build module index
	moduleIndex := discovery.NewModuleIndex(modules)

	// 4. Parse dependencies
	hclParser := parser.NewParser()
	depExtractor := parser.NewDependencyExtractor(hclParser, moduleIndex)
	deps, _ := depExtractor.ExtractAllDependencies()

	// 5. Build dependency graph
	depGraph := graph.BuildFromDependencies(modules, deps)

	// Handle specific module query
	if moduleID != "" {
		return showModuleDependencies(depGraph, moduleIndex, moduleID, showDependents)
	}

	// Handle stats
	if showStats {
		return showGraphStats(depGraph)
	}

	// Generate output based on format
	var output string
	switch graphFormat {
	case "dot":
		output = depGraph.ToDOT()
	case "list":
		output = formatList(depGraph)
	case "levels":
		output, err = formatLevels(depGraph)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown format: %s", graphFormat)
	}

	// Write output
	if graphOutput != "" {
		if err := os.WriteFile(graphOutput, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Graph written to %s\n", graphOutput)
	} else {
		fmt.Print(output)
	}

	return nil
}

func showModuleDependencies(g *graph.DependencyGraph, idx *discovery.ModuleIndex, moduleID string, reverse bool) error {
	if idx.ByID(moduleID) == nil {
		return fmt.Errorf("module not found: %s", moduleID)
	}

	var deps []string
	var label string

	if reverse {
		deps = g.GetAllDependents(moduleID)
		label = "Modules that depend on"
	} else {
		deps = g.GetAllDependencies(moduleID)
		label = "Dependencies of"
	}

	fmt.Printf("%s %s:\n", label, moduleID)
	if len(deps) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, d := range deps {
			fmt.Printf("  - %s\n", d)
		}
	}

	return nil
}

func showGraphStats(g *graph.DependencyGraph) error {
	stats := g.GetStats()

	fmt.Println("Dependency Graph Statistics:")
	fmt.Printf("  Total modules:     %d\n", stats.TotalModules)
	fmt.Printf("  Total edges:       %d\n", stats.TotalEdges)
	fmt.Printf("  Root modules:      %d (no dependencies)\n", stats.RootModules)
	fmt.Printf("  Leaf modules:      %d (no dependents)\n", stats.LeafModules)
	fmt.Printf("  Max depth:         %d\n", stats.MaxDepth)
	fmt.Printf("  Average depth:     %.2f\n", stats.AverageDepth)

	if stats.HasCycles {
		fmt.Printf("  Cycles detected:   %d (WARNING!)\n", stats.CycleCount)

		cycles := g.DetectCycles()
		fmt.Println("\nCycles:")
		for i, cycle := range cycles {
			fmt.Printf("  %d: %s\n", i+1, strings.Join(cycle, " -> "))
		}
	} else {
		fmt.Printf("  Cycles:            none\n")
	}

	return nil
}

func formatList(g *graph.DependencyGraph) string {
	var sb strings.Builder

	sorted, err := g.TopologicalSort()
	if err != nil {
		sb.WriteString(fmt.Sprintf("Error: %s\n", err))
		return sb.String()
	}

	for _, moduleID := range sorted {
		deps := g.GetDependencies(moduleID)
		if len(deps) == 0 {
			sb.WriteString(fmt.Sprintf("%s\n", moduleID))
		} else {
			sb.WriteString(fmt.Sprintf("%s -> %s\n", moduleID, strings.Join(deps, ", ")))
		}
	}

	return sb.String()
}

func formatLevels(g *graph.DependencyGraph) (string, error) {
	levels, err := g.ExecutionLevels()
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	sb.WriteString("Execution Levels (modules at the same level can run in parallel):\n\n")

	for i, level := range levels {
		sb.WriteString(fmt.Sprintf("Level %d:\n", i))
		for _, moduleID := range level {
			sb.WriteString(fmt.Sprintf("  - %s\n", moduleID))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}
