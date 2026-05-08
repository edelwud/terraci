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
	Requirements  BuildRequirements
	PlanEnabled   bool
}

// Build constructs a provider-agnostic IR from the given options.
func Build(opts BuildOptions) (*IR, error) {
	planOnly := opts.Requirements.PlanOnly
	allContributedJobs := collectContributedJobs(opts.Contributions)
	plan, err := buildJobPlan(
		opts.DepGraph, opts.TargetModules, opts.AllModules, opts.ModuleIndex,
	)
	if err != nil {
		return nil, err
	}

	requests := allResourceRequests(opts.Requirements.Resources, allContributedJobs)
	planOutputs := requestedPlanOutputs(plan, requests)

	ir := &IR{
		Levels: buildModuleLevels(plan, opts.PlanEnabled, planOnly, opts.Script, planOutputs),
		Jobs:   buildContributedJobs(allContributedJobs),
	}

	if err := resolvePipelineResources(ir, plan, opts.Requirements.Resources, allContributedJobs); err != nil {
		return nil, err
	}
	if err := ir.Validate(); err != nil {
		return nil, err
	}
	return ir, nil
}

func collectContributedJobs(contributions []*Contribution) []ContributedJob {
	var allContributedJobs []ContributedJob
	for _, c := range contributions {
		if c == nil {
			continue
		}
		allContributedJobs = append(allContributedJobs, c.Jobs...)
	}
	return allContributedJobs
}

func allResourceRequests(required []ResourceRequest, jobs []ContributedJob) []ResourceRequest {
	requests := append([]ResourceRequest(nil), required...)
	for _, job := range jobs {
		requests = append(requests, job.Consumes...)
	}
	return requests
}

func requestedPlanOutputs(plan *JobPlan, requests []ResourceRequest) map[string]PlanOutputs {
	outputs := make(map[string]PlanOutputs, len(plan.TargetModules))
	targets := make(map[string]struct{}, len(plan.TargetModules))
	for _, mod := range plan.TargetModules {
		if mod == nil {
			continue
		}
		targets[mod.RelativePath] = struct{}{}
	}

	for _, request := range requests {
		if !isDetailedPlanResource(request.Kind) {
			continue
		}
		for _, modulePath := range matchingRequestedModulePaths(request, plan.TargetModules, targets) {
			current := outputs[modulePath]
			if request.Kind == ResourceKindPlanText {
				current.Text = true
			}
			if request.Kind == ResourceKindPlanJSON {
				current.JSON = true
			}
			outputs[modulePath] = current
		}
	}
	return outputs
}

func matchingRequestedModulePaths(request ResourceRequest, modules []*discovery.Module, targets map[string]struct{}) []string {
	if request.ModulePath != "" {
		if _, ok := targets[request.ModulePath]; ok {
			return []string{request.ModulePath}
		}
		return nil
	}
	if request.AllModules || (!request.AllProducers && request.Producer == "") {
		paths := make([]string, 0, len(modules))
		for _, mod := range modules {
			if mod != nil {
				paths = append(paths, mod.RelativePath)
			}
		}
		return paths
	}
	return nil
}

func buildModuleLevels(plan *JobPlan, planEnabled, planOnly bool, script ScriptConfig, planOutputs map[string]PlanOutputs) []Level {
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
				mj.Plan = buildPlanJob(plan, mod, env, planOnly, script, planOutputs[mod.RelativePath])
			}

			if !planOnly {
				mj.Apply = buildApplyJob(plan, mod, env, mj.Plan, script)
			}

			level.Modules = append(level.Modules, mj)
		}
		levels = append(levels, level)
	}

	return levels
}

func buildPlanJob(plan *JobPlan, mod *discovery.Module, env map[string]string, planOnly bool, script ScriptConfig, outputs PlanOutputs) *Job {
	planName := JobName(JobKindPlan, mod)
	planOperation, produces, artifact := script.NewPlanOperation(planName, mod.RelativePath, outputs)

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
		Operation:      planOperation,
	}
}

func buildApplyJob(plan *JobPlan, mod *discovery.Module, env map[string]string, planJob *Job, script ScriptConfig) *Job {
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
		Operation:      applyOperation,
	}
}

func buildContributedJobs(contributedJobs []ContributedJob) []Job {
	jobs := make([]Job, 0, len(contributedJobs))
	for _, contributedJob := range contributedJobs {
		job := Job{
			Name:           contributedJob.Name,
			Dependencies:   append([]JobDependency(nil), contributedJob.Dependencies...),
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
