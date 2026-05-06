package pipeline

import (
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
)

// BuildOptions configures the IR builder.
type BuildOptions struct {
	DepGraph          *graph.DependencyGraph
	TargetModules     []*discovery.Module
	AllModules        []*discovery.Module
	ModuleIndex       *discovery.ModuleIndex
	Script            ScriptConfig
	Contributions     []*Contribution
	RequiredResources []ResourceRequest
	PlanEnabled       bool
	PlanOnly          bool
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

	allSteps, allContributedJobs := collectContributionParts(opts.Contributions)
	script := effectiveScript(opts.Script, opts.RequiredResources, allContributedJobs)

	ir := &IR{
		Levels: buildModuleLevels(plan, opts.PlanEnabled, opts.PlanOnly, script, allSteps),
		Jobs:   buildContributedJobs(allContributedJobs),
	}

	if err := resolvePipelineResources(ir, plan, opts.RequiredResources, allContributedJobs); err != nil {
		return nil, err
	}
	if err := ir.Validate(); err != nil {
		return nil, err
	}
	return ir, nil
}

func collectContributionParts(contributions []*Contribution) ([]Step, []ContributedJob) {
	var allSteps []Step
	var allContributedJobs []ContributedJob
	for _, c := range contributions {
		if c == nil {
			continue
		}
		allSteps = append(allSteps, c.Steps...)
		allContributedJobs = append(allContributedJobs, c.Jobs...)
	}
	return allSteps, allContributedJobs
}

func effectiveScript(script ScriptConfig, required []ResourceRequest, jobs []ContributedJob) ScriptConfig {
	if requestsRequireDetailedPlan(required, jobs) {
		script.DetailedPlan = true
	}
	return script
}

func buildModuleLevels(plan *JobPlan, planEnabled, planOnly bool, script ScriptConfig, steps []Step) []Level {
	levels := make([]Level, 0, len(plan.ExecutionLevels))
	for levelIdx, moduleIDs := range plan.ExecutionLevels {
		level := Level{Index: levelIdx}
		for _, moduleID := range moduleIDs {
			mod := plan.ModuleIndex.ByID(moduleID)
			if mod == nil {
				continue
			}

			env := ModuleEnvVars(mod)
			mj := ModuleJobs{Module: mod}

			if planEnabled {
				mj.Plan = buildPlanJob(plan, mod, env, planOnly, script, steps)
			}

			if !planOnly {
				mj.Apply = buildApplyJob(plan, mod, env, mj.Plan, script, steps)
			}

			level.Modules = append(level.Modules, mj)
		}
		levels = append(levels, level)
	}

	return levels
}

func buildPlanJob(plan *JobPlan, mod *discovery.Module, env map[string]string, planOnly bool, script ScriptConfig, steps []Step) *Job {
	planName := JobName(JobKindPlan, mod)
	planOperation, produces, artifact := script.NewPlanOperation(planName, mod.RelativePath)

	var deps []JobDependency
	if planOnly {
		deps = controlDependencies(ResolveDependencyNames(mod, JobKindPlan, plan.Subgraph, plan.ModuleIndex))
	} else {
		deps = controlDependencies(ResolveDependencyNames(mod, JobKindApply, plan.Subgraph, plan.ModuleIndex))
	}

	return &Job{
		Name:           planName,
		Module:         mod,
		Env:            env,
		Dependencies:   deps,
		OutputArtifact: artifact,
		Produces:       produces,
		Steps:          filterSteps(steps, PhasePrePlan, PhasePostPlan),
		Operation:      planOperation,
	}
}

func buildApplyJob(plan *JobPlan, mod *discovery.Module, env map[string]string, planJob *Job, script ScriptConfig, steps []Step) *Job {
	applyOperation := script.NewApplyOperation(mod.RelativePath)
	applyDeps := controlDependencies(ResolveDependencyNames(mod, JobKindApply, plan.Subgraph, plan.ModuleIndex))
	var consumes []ResourceSpec
	var inputArtifacts []Artifact

	if planJob != nil {
		applyDeps = append([]JobDependency{{
			Job:       planJob.Name,
			Artifacts: true,
		}}, applyDeps...)
		consumes = append(consumes,
			PlanResource(ResourceKindPlanBinary, mod.RelativePath, applyOperation.Terraform.PlanFile),
		)
		inputArtifacts = append(inputArtifacts, planJob.OutputArtifact)
	}

	return &Job{
		Name:           JobName(JobKindApply, mod),
		Module:         mod,
		Env:            env,
		Dependencies:   applyDeps,
		InputArtifacts: inputArtifacts,
		Consumes:       consumes,
		Steps:          filterSteps(steps, PhasePreApply, PhasePostApply),
		Operation:      applyOperation,
	}
}

func buildContributedJobs(contributedJobs []ContributedJob) []Job {
	jobs := make([]Job, 0, len(contributedJobs))
	for _, contributedJob := range contributedJobs {
		job := Job{
			Name:           contributedJob.Name,
			Phase:          contributedJob.Phase,
			Dependencies:   contributedJobDependencies(contributedJob, contributedJobs),
			OutputArtifact: resultArtifactFromResources(contributedJob.Name, contributedJob.Produces),
			Produces:       append([]ResourceSpec(nil), contributedJob.Produces...),
			AllowFailure:   contributedJob.AllowFailure,
			Operation: Operation{
				Type:     OperationTypeCommands,
				Commands: append([]string(nil), contributedJob.Commands...),
			},
		}
		jobs = append(jobs, job)
	}
	return jobs
}

func contributedJobDependencies(job ContributedJob, allJobs []ContributedJob) []JobDependency {
	if job.Phase != PhaseFinalize {
		return nil
	}

	deps := make([]JobDependency, 0)
	for _, other := range allJobs {
		if other.Name != job.Name && other.Phase != PhaseFinalize {
			deps = mergeJobDependency(deps, JobDependency{Job: other.Name})
		}
	}
	return deps
}

func resolvePipelineResources(ir *IR, plan *JobPlan, required []ResourceRequest, contributedJobs []ContributedJob) error {
	resources, err := buildResourceIndex(ir)
	if err != nil {
		return err
	}

	allowEmptyModuleResources := len(plan.TargetModules) == 0
	if _, _, err := resolveResourceRequests(required, resources, "pipeline required resources", allowEmptyModuleResources); err != nil {
		return err
	}

	for i := range ir.Jobs {
		consumes, artifacts, deps, err := resolveResourceRequestsForJob(contributedJobs[i].Consumes, resources, ir.Jobs[i].Name, allowEmptyModuleResources)
		if err != nil {
			return err
		}
		ir.Jobs[i].Consumes = consumes
		ir.Jobs[i].InputArtifacts = artifacts
		for _, dep := range deps {
			ir.Jobs[i].Dependencies = mergeJobDependency(ir.Jobs[i].Dependencies, dep)
		}
	}
	return nil
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
		if c == nil {
			continue
		}
		if len(c.Jobs) > 0 {
			return true
		}
	}
	return false
}
