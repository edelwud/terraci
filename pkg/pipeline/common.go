package pipeline

import (
	"fmt"
	"strings"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
)

// JobPlan contains prepared data for pipeline generation.
type JobPlan struct {
	TargetModules       []*discovery.Module
	TargetSet           map[string]bool
	ExecutionLevels     [][]string
	Subgraph            *graph.DependencyGraph
	ModuleIndex         *discovery.ModuleIndex
	HasContributedJobs  bool
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

// BuildDryRunResult creates a DryRunResult from a job plan.
func BuildDryRunResult(plan *JobPlan, totalModules int, planEnabled bool) *DryRunResult {
	jobCount := 0
	for _, level := range plan.ExecutionLevels {
		jobCount += len(level)
		if planEnabled {
			jobCount += len(level)
		}
	}

	stageCount := len(plan.ExecutionLevels)
	if plan.HasContributedJobs {
		jobCount++
		stageCount++
	}

	return &DryRunResult{
		TotalModules:    totalModules,
		AffectedModules: len(plan.TargetModules),
		Stages:          stageCount,
		Jobs:            jobCount,
		ExecutionOrder:  plan.ExecutionLevels,
	}
}
