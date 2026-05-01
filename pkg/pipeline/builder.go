package pipeline

import (
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
)

// BuildOptions configures the IR builder.
type BuildOptions struct {
	DepGraph      *graph.DependencyGraph
	TargetModules []*discovery.Module
	AllModules    []*discovery.Module
	ModuleIndex   *discovery.ModuleIndex
	Script        ScriptConfig
	Contributions []*Contribution
	PlanEnabled   bool
	PlanOnly      bool
}

// Build constructs a provider-agnostic IR from the given options.
func Build(opts BuildOptions) (*IR, error) {
	plan, err := buildJobPlan(
		opts.DepGraph, opts.TargetModules, opts.AllModules, opts.ModuleIndex,
		hasContributedJobs(opts.Contributions), opts.PlanEnabled,
	)
	if err != nil {
		return nil, err
	}

	// Collect all steps and jobs from contributions
	var allSteps []Step
	var allContributedJobs []ContributedJob
	for _, c := range opts.Contributions {
		allSteps = append(allSteps, c.Steps...)
		allContributedJobs = append(allContributedJobs, c.Jobs...)
	}

	ir := &IR{}

	// Build levels with module jobs
	for levelIdx, moduleIDs := range plan.ExecutionLevels {
		level := Level{Index: levelIdx}
		for _, moduleID := range moduleIDs {
			mod := plan.ModuleIndex.ByID(moduleID)
			if mod == nil {
				continue
			}

			env := ModuleEnvVars(mod)
			mj := ModuleJobs{Module: mod}

			// Plan job
			if opts.PlanEnabled {
				planOperation, artifactPaths := opts.Script.NewPlanOperation(mod.RelativePath)
				planName := JobName("plan", mod)

				// Resolve plan dependencies
				var planDeps []string
				if opts.PlanOnly {
					planDeps = ResolveDependencyNames(mod, "plan", plan.TargetSet, plan.Subgraph, plan.ModuleIndex)
				} else {
					planDeps = ResolveDependencyNames(mod, "apply", plan.TargetSet, plan.Subgraph, plan.ModuleIndex)
				}

				mj.Plan = &Job{
					Name:          planName,
					Type:          JobTypePlan,
					Module:        mod,
					Env:           env,
					Dependencies:  planDeps,
					ArtifactPaths: artifactPaths,
					Steps:         filterSteps(allSteps, PhasePrePlan, PhasePostPlan),
					Operation:     planOperation,
				}
			}

			// Apply job
			if !opts.PlanOnly {
				applyOperation := opts.Script.NewApplyOperation(mod.RelativePath)
				applyName := JobName("apply", mod)

				applyDeps := ResolveDependencyNames(mod, "apply", plan.TargetSet, plan.Subgraph, plan.ModuleIndex)

				// Apply depends on its own plan job
				if mj.Plan != nil {
					applyDeps = append([]string{mj.Plan.Name}, applyDeps...)
				}

				mj.Apply = &Job{
					Name:         applyName,
					Type:         JobTypeApply,
					Module:       mod,
					Env:          env,
					Dependencies: applyDeps,
					Steps:        filterSteps(allSteps, PhasePreApply, PhasePostApply),
					Operation:    applyOperation,
				}
			}

			level.Modules = append(level.Modules, mj)
		}
		ir.Levels = append(ir.Levels, level)
	}

	// Contributed jobs.
	planNames := ir.AllPlanNames()

	// First pass: create all jobs so we know their names
	irJobs := make([]Job, 0, len(allContributedJobs))
	for _, cj := range allContributedJobs {
		job := Job{
			Name:          cj.Name,
			Phase:         cj.Phase,
			ArtifactPaths: cj.ArtifactPaths,
			AllowFailure:  cj.AllowFailure,
			Operation: Operation{
				Type:     OperationTypeCommands,
				Commands: append([]string(nil), cj.Commands...),
			},
		}

		deps := make([]string, 0)
		if cj.DependsOnPlan {
			deps = append(deps, planNames...)
		}
		if cj.Phase == PhaseFinalize {
			// Finalize jobs run after the earlier contribution phases. Multiple
			// finalize jobs may run together unless a contributor adds an explicit edge.
			for _, other := range allContributedJobs {
				if other.Name != cj.Name && other.Phase != PhaseFinalize {
					deps = append(deps, other.Name)
				}
			}
		}
		job.Dependencies = deps
		irJobs = append(irJobs, job)
	}
	ir.Jobs = irJobs

	if err := ir.Validate(); err != nil {
		return nil, err
	}
	return ir, nil
}

// filterSteps returns steps matching any of the given phases.
func filterSteps(steps []Step, phases ...Phase) []Step {
	phaseSet := make(map[Phase]bool, len(phases))
	for _, p := range phases {
		phaseSet[p] = true
	}
	var result []Step
	for _, s := range steps {
		if phaseSet[s.Phase] {
			result = append(result, s)
		}
	}
	return result
}

// hasContributedJobs checks if any contributions contain jobs.
func hasContributedJobs(contributions []*Contribution) bool {
	for _, c := range contributions {
		if len(c.Jobs) > 0 {
			return true
		}
	}
	return false
}
