// Package graphflow owns dependency graph command orchestration.
package graphflow

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/edelwud/terraci/cmd/terraci/internal/projectflow"
	"github.com/edelwud/terraci/cmd/terraci/internal/runflow"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/graph"
)

// Format is a supported graph output format.
type Format string

const (
	FormatDOT      Format = "dot"
	FormatPlantUML Format = "plantuml"
	FormatList     Format = "list"
	FormatLevels   Format = "levels"
)

// ParseFormat parses a graph format flag.
func ParseFormat(raw string) (Format, error) {
	switch Format(raw) {
	case "", FormatDOT:
		return FormatDOT, nil
	case FormatPlantUML:
		return FormatPlantUML, nil
	case FormatList:
		return FormatList, nil
	case FormatLevels:
		return FormatLevels, nil
	default:
		return "", fmt.Errorf("unknown format: %s", raw)
	}
}

// Runtime contains immutable dependencies needed to render a graph.
type Runtime struct {
	project projectflow.Runtime
}

// NewRuntime creates a graph runtime from prepared command state.
func NewRuntime(prepared *runflow.Prepared) Runtime {
	return Runtime{project: projectflow.NewRuntime(prepared)}
}

// Request describes one graph command request.
type Request struct {
	Filters        filter.Flags
	Format         string
	ModuleID       string
	ShowDependents bool
	ShowStats      bool
}

// Result contains the graph command outcome.
type Result struct {
	Project     *projectflow.Result
	Output      string
	Stats       *StatsResult
	ModuleCount int
}

// StatsResult contains graph statistics ready for presentation.
type StatsResult struct {
	Scope  string
	Stats  graph.Stats
	Cycles [][]string
}

// Run scans the project, optionally scopes the graph, and renders the requested output.
func Run(ctx context.Context, runtime Runtime, req Request) (*Result, error) {
	project, err := projectflow.Run(ctx, runtime.project, projectflow.Request{Filters: req.Filters})
	if err != nil {
		return nil, err
	}

	depGraph := project.Workflow.Graph
	libraries := project.Workflow.Libraries.Modules
	if req.ModuleID != "" {
		depGraph, err = depGraph.ScopeToModule(req.ModuleID, req.ShowDependents)
		if err != nil {
			return nil, err
		}
		// Scoped graphs describe an executable subtree; hide library nodes to
		// keep the visualization focused.
		libraries = nil
	}

	result := &Result{
		Project:     project,
		ModuleCount: len(project.Workflow.Filtered.Modules),
	}
	if req.ShowStats {
		result.Stats = &StatsResult{
			Scope:  req.ModuleID,
			Stats:  depGraph.GetStats(),
			Cycles: depGraph.DetectCycles(),
		}
		return result, nil
	}

	format, err := ParseFormat(req.Format)
	if err != nil {
		return nil, err
	}
	output, err := Render(depGraph, libraries, format)
	if err != nil {
		return nil, err
	}
	result.Output = output
	return result, nil
}

// Render renders a dependency graph in a supported format.
func Render(g *graph.DependencyGraph, libraries []*discovery.Module, format Format) (string, error) {
	switch format {
	case FormatDOT:
		return g.ToDOTWithLibraries(libraries), nil
	case FormatPlantUML:
		return g.ToPlantUML(), nil
	case FormatList:
		return formatList(g, libraries)
	case FormatLevels:
		return formatLevels(g)
	default:
		return "", fmt.Errorf("unknown format: %s", format)
	}
}

func formatList(g *graph.DependencyGraph, libraries []*discovery.Module) (string, error) {
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

	if len(libraries) > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("[library_modules]\n")
		ids := make([]string, 0, len(libraries))
		for _, module := range libraries {
			ids = append(ids, module.RelativePath)
		}
		sort.Strings(ids)
		for _, id := range ids {
			fmt.Fprintf(&sb, "  %s\n", id)
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
