package pipeline

import (
	"fmt"
	"strings"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
)

// jobPlan contains prepared data for pipeline generation. The Subgraph is
// already filtered to the target module set, so callers do not need to track
// "in target" IDs separately — Subgraph.GetDependencies returns only
// target-included dependencies.
type jobPlan struct {
	targetModules []*discovery.Module
	moduleOrder   []string
	subgraph      *graph.DependencyGraph
	moduleIndex   *discovery.ModuleIndex
}

// prepareModuleGraph prepares the module graph from target modules. It is an
// internal detail of Build() and is not part of the public package API.
func prepareModuleGraph(
	depGraph *graph.DependencyGraph,
	targetModules, allModules []*discovery.Module,
	moduleIndex *discovery.ModuleIndex,
) (*jobPlan, error) {
	if len(targetModules) == 0 {
		targetModules = allModules
	}

	moduleIDs := make([]string, len(targetModules))
	for i, m := range targetModules {
		moduleIDs[i] = m.ID()
	}

	subgraph := depGraph.Subgraph(moduleIDs)

	moduleOrder, err := subgraph.TopologicalSort()
	if err != nil {
		return nil, fmt.Errorf("failed to calculate module order: %w", err)
	}

	return &jobPlan{
		targetModules: targetModules,
		moduleOrder:   moduleOrder,
		subgraph:      subgraph,
		moduleIndex:   moduleIndex,
	}, nil
}

// JobName generates a safe job name from module path and job kind.
// Only Plan/Apply kinds are supported here; contributed jobs carry their
// own name (assigned by the contributor).
func JobName(kind JobKind, module *discovery.Module) string {
	prefix := kind.NamePrefix()
	if prefix == "" {
		return strings.ReplaceAll(module.ID(), "/", "-")
	}
	name := strings.ReplaceAll(module.ID(), "/", "-")
	return fmt.Sprintf("%s-%s", prefix, name)
}

// ResolveDependencyNames returns job names for a module's dependencies in the
// supplied subgraph. The subgraph is expected to already be scoped to the
// target module set; only modules present in moduleIndex are emitted.
func ResolveDependencyNames(
	module *discovery.Module,
	kind JobKind,
	subgraph *graph.DependencyGraph,
	moduleIndex *discovery.ModuleIndex,
) []string {
	deps := subgraph.GetDependencies(module.ID())
	names := make([]string, 0, len(deps))
	for _, depID := range deps {
		depModule := moduleIndex.ByID(depID)
		if depModule == nil {
			continue
		}
		names = append(names, JobName(kind, depModule))
	}
	return names
}

// DryRun summarizes what the IR would produce. The caller passes the total
// number of source modules (which can exceed the IR's affected count when
// filters are active).
func (ir *IR) DryRun(totalModules int) *DryRunResult {
	if ir == nil {
		return &DryRunResult{TotalModules: totalModules}
	}

	jobGroups := make([][]string, 0)
	stages := 0
	if groups, err := Schedule(ir); err == nil {
		stages = len(groups)
		jobGroups = make([][]string, 0, len(groups))
		for _, group := range groups {
			names := make([]string, 0, len(group.Jobs))
			for _, job := range group.Jobs {
				if job != nil {
					names = append(names, job.Name)
				}
			}
			jobGroups = append(jobGroups, names)
		}
	}

	return &DryRunResult{
		TotalModules:    totalModules,
		AffectedModules: ir.ModuleCount(),
		Stages:          stages,
		Jobs:            len(ir.Jobs),
		JobGroups:       jobGroups,
	}
}
