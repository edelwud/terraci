package pipeline

import (
	"fmt"
	"strings"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
)

// JobPlan contains prepared data for pipeline generation.
type JobPlan struct {
	TargetModules      []*discovery.Module
	TargetSet          map[string]bool
	ExecutionLevels    [][]string
	Subgraph           *graph.DependencyGraph
	ModuleIndex        *discovery.ModuleIndex
	HasContributedJobs bool
}

// BuildJobPlan prepares the execution plan from target modules.
func BuildJobPlan(
	depGraph *graph.DependencyGraph,
	targetModules, allModules []*discovery.Module,
	moduleIndex *discovery.ModuleIndex,
	hasContributedJobs, planEnabled bool,
) (*JobPlan, error) {
	if len(targetModules) == 0 {
		targetModules = allModules
	}

	moduleIDs := make([]string, len(targetModules))
	targetSet := make(map[string]bool, len(targetModules))
	for i, m := range targetModules {
		moduleIDs[i] = m.ID()
		targetSet[m.ID()] = true
	}

	subgraph := depGraph.Subgraph(moduleIDs)

	levels, err := subgraph.ExecutionLevels()
	if err != nil {
		return nil, fmt.Errorf("failed to calculate execution levels: %w", err)
	}

	return &JobPlan{
		TargetModules:      targetModules,
		TargetSet:          targetSet,
		ExecutionLevels:    levels,
		Subgraph:           subgraph,
		ModuleIndex:        moduleIndex,
		HasContributedJobs: hasContributedJobs && planEnabled,
	}, nil
}

// JobName generates a safe job name from module path and job type.
func JobName(jobType string, module *discovery.Module) string {
	name := strings.ReplaceAll(module.ID(), "/", "-")
	return fmt.Sprintf("%s-%s", jobType, name)
}

// ResolveDependencyNames returns job names for a module's dependencies within the target set.
func ResolveDependencyNames(
	module *discovery.Module,
	jobType string,
	targetSet map[string]bool,
	depGraph *graph.DependencyGraph,
	moduleIndex *discovery.ModuleIndex,
) []string {
	deps := depGraph.GetDependencies(module.ID())
	names := make([]string, 0, len(deps))
	for _, depID := range deps {
		if !targetSet[depID] {
			continue
		}
		depModule := moduleIndex.ByID(depID)
		if depModule == nil {
			continue
		}
		names = append(names, JobName(jobType, depModule))
	}
	return names
}

// BuildDryRunResult creates a DryRunResult from the canonical pipeline IR.
func BuildDryRunResult(ir *IR, totalModules int) *DryRunResult {
	if ir == nil {
		return &DryRunResult{TotalModules: totalModules}
	}

	executionOrder := make([][]string, 0, len(ir.Levels))
	jobCount := len(ir.Jobs)
	affectedModules := 0
	for _, level := range ir.Levels {
		levelOrder := make([]string, 0, len(level.Modules))
		for _, moduleJobs := range level.Modules {
			if moduleJobs.Module != nil {
				levelOrder = append(levelOrder, moduleJobs.Module.ID())
				affectedModules++
			}
			if moduleJobs.Plan != nil {
				jobCount++
			}
			if moduleJobs.Apply != nil {
				jobCount++
			}
		}
		executionOrder = append(executionOrder, levelOrder)
	}

	contributedPhases := make(map[Phase]struct{}, len(ir.Jobs))
	for _, ref := range ir.JobRefs() {
		if ref.Kind == JobKindContributed && ref.Job != nil {
			contributedPhases[ref.Job.Phase] = struct{}{}
		}
	}

	return &DryRunResult{
		TotalModules:    totalModules,
		AffectedModules: affectedModules,
		Stages:          len(ir.Levels) + len(contributedPhases),
		Jobs:            jobCount,
		ExecutionOrder:  executionOrder,
	}
}
