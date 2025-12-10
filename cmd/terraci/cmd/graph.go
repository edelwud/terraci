package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/filter"
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

  # Filter graph to a specific module and its dependencies
  terraci graph --module platform/stage/eu-central-1/vpc --format levels

  # Filter graph to show what depends on a module
  terraci graph --module platform/stage/eu-central-1/vpc --dependents --stats

  # Generate DOT for module subgraph
  terraci graph --module platform/stage/eu-central-1/vpc -o module.dot`,
	RunE: runGraph,
}

func init() {
	rootCmd.AddCommand(graphCmd)

	graphCmd.Flags().StringVarP(&graphFormat, "format", "f", "dot", "output format: dot, list, levels")
	graphCmd.Flags().StringVarP(&graphOutput, "output", "o", "", "output file (default: stdout)")
	graphCmd.Flags().BoolVar(&showStats, "stats", false, "show graph statistics")
	graphCmd.Flags().StringVarP(&moduleID, "module", "m", "", "filter graph to specific module and its dependencies")
	graphCmd.Flags().BoolVar(&showDependents, "dependents", false, "include dependents instead of dependencies when using --module")

	// Reuse filter flags from generate
	graphCmd.Flags().StringArrayVarP(&excludes, "exclude", "x", nil, "glob patterns to exclude modules")
	graphCmd.Flags().StringArrayVarP(&includes, "include", "i", nil, "glob patterns to include modules")
}

func runGraph(_ *cobra.Command, _ []string) error {
	// 1. Discover modules
	log.WithField("dir", workDir).Debug("scanning for modules")
	scanner := discovery.NewScanner(workDir)
	scanner.MinDepth = cfg.Structure.MinDepth
	scanner.MaxDepth = cfg.Structure.MaxDepth

	modules, err := scanner.Scan()
	if err != nil {
		return fmt.Errorf("failed to scan modules: %w", err)
	}

	log.WithField("count", len(modules)).Debug("modules discovered")

	if len(modules) == 0 {
		return fmt.Errorf("no modules found in %s", workDir)
	}

	// 2. Apply filters
	allExcludes := append([]string{}, cfg.Exclude...)
	allExcludes = append(allExcludes, excludes...)
	allIncludes := append([]string{}, cfg.Include...)
	allIncludes = append(allIncludes, includes...)
	globFilter := filter.NewGlobFilter(allExcludes, allIncludes)
	modules = globFilter.FilterModules(modules)

	// 3. Build module index
	moduleIndex := discovery.NewModuleIndex(modules)

	// 4. Parse dependencies
	log.Debug("parsing dependencies")
	hclParser := parser.NewParser()
	depExtractor := parser.NewDependencyExtractor(hclParser, moduleIndex)
	deps, _ := depExtractor.ExtractAllDependencies()

	// 5. Build dependency graph
	log.Debug("building dependency graph")
	depGraph := graph.BuildFromDependencies(modules, deps)

	// Handle specific module filtering - creates a subgraph scoped to the module
	if moduleID != "" {
		if moduleIndex.ByID(moduleID) == nil {
			return fmt.Errorf("module not found: %s", moduleID)
		}

		var moduleIDs []string
		if showDependents {
			// Include the module and all modules that depend on it
			moduleIDs = append([]string{moduleID}, depGraph.GetAllDependents(moduleID)...)
			log.WithField("module", moduleID).WithField("count", len(moduleIDs)-1).Debug("filtering to module and its dependents")
		} else {
			// Include the module and all its dependencies
			moduleIDs = append([]string{moduleID}, depGraph.GetAllDependencies(moduleID)...)
			log.WithField("module", moduleID).WithField("count", len(moduleIDs)-1).Debug("filtering to module and its dependencies")
		}

		// Create subgraph with only the relevant modules
		depGraph = depGraph.Subgraph(moduleIDs)
	}

	// Handle stats
	if showStats {
		return showGraphStats(depGraph, moduleID)
	}

	// Generate output based on format
	log.WithField("format", graphFormat).Debug("generating output")

	// For file output, use string formatting
	if graphOutput != "" {
		var output string
		switch graphFormat {
		case "dot":
			output = depGraph.ToDOT()
		case "list":
			output = formatListString(depGraph)
		case "levels":
			output, err = formatLevelsString(depGraph)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown format: %s", graphFormat)
		}

		if err := os.WriteFile(graphOutput, []byte(output), 0o600); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		log.WithField("file", graphOutput).Info("graph written")
		return nil
	}

	// For stdout, use logger for list/levels, raw output for dot
	switch graphFormat {
	case "dot":
		fmt.Print(depGraph.ToDOT())
	case "list":
		return printList(depGraph)
	case "levels":
		return printLevels(depGraph)
	default:
		return fmt.Errorf("unknown format: %s", graphFormat)
	}

	return nil
}

func showGraphStats(g *graph.DependencyGraph, scopeModule string) error {
	stats := g.GetStats()

	if scopeModule != "" {
		log.WithField("scope", scopeModule).Info("dependency graph statistics")
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
		cycles := g.DetectCycles()
		log.IncreasePadding()
		for i, cycle := range cycles {
			log.WithField("cycle", i+1).WithField("path", strings.Join(cycle, " -> ")).Warn("cycle")
		}
		log.DecreasePadding()
	} else {
		log.Info("no cycles")
	}
	log.DecreasePadding()

	return nil
}

// printList outputs the dependency list using the logger
func printList(g *graph.DependencyGraph) error {
	sorted, err := g.TopologicalSort()
	if err != nil {
		return err
	}

	log.Info("module dependencies (topological order)")
	log.IncreasePadding()
	for _, moduleID := range sorted {
		deps := g.GetDependencies(moduleID)
		if len(deps) == 0 {
			log.Info(moduleID)
		} else {
			log.WithField("deps", strings.Join(deps, ", ")).Info(moduleID)
		}
	}
	log.DecreasePadding()

	return nil
}

// printLevels outputs the execution levels using the logger
func printLevels(g *graph.DependencyGraph) error {
	levels, err := g.ExecutionLevels()
	if err != nil {
		return err
	}

	log.Info("execution levels (modules at the same level can run in parallel)")
	for i, level := range levels {
		log.WithField("level", i).WithField("count", len(level)).Info("level")
		log.IncreasePadding()
		for _, moduleID := range level {
			log.Info(moduleID)
		}
		log.DecreasePadding()
	}

	return nil
}

// formatListString returns the dependency list as a string (for file output)
func formatListString(g *graph.DependencyGraph) string {
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

// formatLevelsString returns the execution levels as a string (for file output)
func formatLevelsString(g *graph.DependencyGraph) (string, error) {
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
