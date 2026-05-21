package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/cmd/terraci/internal/graphflow"
	"github.com/edelwud/terraci/cmd/terraci/internal/runflow"
	"github.com/edelwud/terraci/pkg/filter"
)

func newGraphCmd() *cobra.Command {
	var (
		graphFormat    string
		graphOutput    string
		showStats      bool
		moduleID       string
		showDependents bool
	)
	ff := &filter.Flags{}

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
			prepared, err := runflow.FromContext(cmd.Context())
			if err != nil {
				return err
			}
			result, err := graphflow.Run(cmd.Context(), graphflow.NewRuntime(prepared), graphflow.Request{
				Filters:        *ff,
				Format:         graphFormat,
				ModuleID:       moduleID,
				ShowDependents: showDependents,
				ShowStats:      showStats,
			})
			if err != nil {
				return err
			}

			log.WithField("count", result.ModuleCount).Debug("modules after filtering")
			if showStats {
				logGraphStats(result.Stats)
				return nil
			}

			return writeGraphOutput(result.Output, graphOutput)
		},
	}
	runflow.MarkCommand(cmd, runflow.CommandPolicy{SkipPreflight: true})

	cmd.Flags().StringVarP(&graphFormat, "format", "F", "dot", "output format: dot, plantuml, list, levels")
	cmd.Flags().StringVarP(&graphOutput, "output", "o", "", "output file (default: stdout)")
	cmd.Flags().BoolVar(&showStats, "stats", false, "show graph statistics")
	cmd.Flags().StringVarP(&moduleID, "module", "m", "", "filter to specific module")
	cmd.Flags().BoolVar(&showDependents, "dependents", false, "show dependents instead of dependencies (with --module)")
	registerFilterFlags(cmd, ff)

	return cmd
}

func writeGraphOutput(output, outputFile string) error {
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

func logGraphStats(result *graphflow.StatsResult) {
	if result == nil {
		return
	}
	stats := result.Stats
	if result.Scope != "" {
		log.WithField("scope", result.Scope).Info("dependency graph statistics")
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
		for i, cycle := range result.Cycles {
			log.WithField("cycle", i+1).WithField("path", strings.Join(cycle, " → ")).Warn("cycle")
		}
		log.DecreasePadding()
	} else {
		log.Info("no cycles ✓")
	}

	log.DecreasePadding()
}
