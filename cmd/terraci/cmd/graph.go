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

  # Show dependencies for a specific module
  terraci graph --module platform/stage/eu-central-1/vpc

  # Show what depends on a module
  terraci graph --module platform/stage/eu-central-1/vpc --dependents`,
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

	// Handle specific module query
	if moduleID != "" {
		return showModuleDependencies(depGraph, moduleIndex, moduleID, showDependents)
	}

	// Handle stats
	if showStats {
		return showGraphStats(depGraph)
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

func showModuleDependencies(g *graph.DependencyGraph, idx *discovery.ModuleIndex, moduleID string, reverse bool) error {
	if idx.ByID(moduleID) == nil {
		return fmt.Errorf("module not found: %s", moduleID)
	}

	var deps []string
	var label string

	if reverse {
		deps = g.GetAllDependents(moduleID)
		label = "modules that depend on"
	} else {
		deps = g.GetAllDependencies(moduleID)
		label = "dependencies of"
	}

	log.WithField("module", moduleID).Info(label)
	if len(deps) == 0 {
		log.Info("(none)")
	} else {
		log.IncreasePadding()
		for _, d := range deps {
			log.Info(d)
		}
		log.DecreasePadding()
	}

	return nil
}

func showGraphStats(g *graph.DependencyGraph) error {
	stats := g.GetStats()

	log.Info("dependency graph statistics")
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
