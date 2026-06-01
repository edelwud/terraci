package pipeline

import (
	"fmt"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
)

type projectIRBuildInput struct {
	DepGraph      *graph.DependencyGraph
	TargetModules []*discovery.Module
	AllModules    []*discovery.Module
	ModuleIndex   *discovery.ModuleIndex
	Script        ScriptConfig
	Contributions []*Contribution
	Intent        BuildIntent
}

func buildProjectIR(opts projectIRBuildInput) (*IR, error) {
	allContributedJobs := collectContributedJobs(opts.Contributions)
	plan, err := prepareModuleGraph(
		opts.DepGraph, opts.TargetModules, opts.AllModules, opts.ModuleIndex,
	)
	if err != nil {
		return nil, err
	}

	required := opts.Intent.ResourceRequests()
	requests := allResourceRequests(required, allContributedJobs)
	if err := validateBuildResourceRequests(required, opts.Contributions); err != nil {
		return nil, err
	}
	planOutputs := requestedPlanOutputs(plan, requests)

	ir := &IR{
		jobs: buildJobs(plan, opts.Intent, opts.Script, planOutputs, requests, allContributedJobs),
	}

	if err := resolvePipelineResources(ir, plan, required, allContributedJobs); err != nil {
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
		allContributedJobs = append(allContributedJobs, c.Jobs()...)
	}
	return allContributedJobs
}

func allResourceRequests(required []ResourceRequest, jobs []ContributedJob) []ResourceRequest {
	requests := append([]ResourceRequest(nil), required...)
	for _, job := range jobs {
		requests = append(requests, job.Consumes()...)
	}
	return requests
}

func validateBuildResourceRequests(required []ResourceRequest, contributions []*Contribution) error {
	for i, request := range required {
		if err := validateResourceRequest(request); err != nil {
			return fmt.Errorf("requirements.resources[%d]: %w", i, err)
		}
	}
	for contributionIdx, contribution := range contributions {
		if contribution == nil {
			continue
		}
		for jobIdx, job := range contribution.Jobs() {
			for reqIdx, request := range job.Consumes() {
				if err := validateResourceRequest(request); err != nil {
					return fmt.Errorf("contributions[%d].jobs[%d].consumes[%d]: %w", contributionIdx, jobIdx, reqIdx, err)
				}
			}
		}
	}
	return nil
}

func requestedPlanOutputs(plan *jobPlan, requests []ResourceRequest) map[string]PlanOutputs {
	outputs := make(map[string]PlanOutputs, len(plan.targetModules))
	targets := make(map[string]struct{}, len(plan.targetModules))
	for _, mod := range plan.targetModules {
		if mod == nil {
			continue
		}
		targets[mod.ID()] = struct{}{}
	}

	for _, request := range requests {
		if !isDetailedPlanResource(request.kind) {
			continue
		}
		for _, modulePath := range matchingRequestedModulePaths(request, plan.targetModules, targets) {
			current := outputs[modulePath]
			if request.kind == ResourceKindPlanText {
				current.Text = true
			}
			if request.kind == ResourceKindPlanJSON {
				current.JSON = true
			}
			outputs[modulePath] = current
		}
	}
	return outputs
}

func matchingRequestedModulePaths(request ResourceRequest, modules []*discovery.Module, targets map[string]struct{}) []string {
	switch request.selector.scope {
	case ResourceScopeModule:
		if _, ok := targets[request.selector.modulePath]; ok {
			return []string{request.selector.modulePath}
		}
		return nil
	case ResourceScopeAllModules:
		paths := make([]string, 0, len(modules))
		for _, mod := range modules {
			if mod != nil {
				paths = append(paths, mod.ID())
			}
		}
		return paths
	case ResourceScopeAllProducers, ResourceScopeProducer:
		return nil
	default:
		return nil
	}
}

func buildJobs(plan *jobPlan, intent BuildIntent, script ScriptConfig, planOutputs map[string]PlanOutputs, requests []ResourceRequest, contributedJobs []ContributedJob) []Job {
	jobs := buildModuleJobs(plan, intent, script, planOutputs, requests)
	jobs = append(jobs, buildContributedJobs(contributedJobs)...)
	return jobs
}

func buildModuleJobs(plan *jobPlan, intent BuildIntent, script ScriptConfig, planOutputs map[string]PlanOutputs, requests []ResourceRequest) []Job {
	jobs := make([]Job, 0, len(plan.targetModules)*2)
	for _, moduleID := range plan.moduleOrder {
		mod := plan.moduleIndex.ByID(moduleID)
		if mod == nil {
			continue
		}
		modulePath := mod.ID()

		env := TerraformJobEnv(script.TerraformEnv(), mod)
		var planJob *Job

		if moduleNeedsPlanJob(modulePath, intent, requests) {
			job := buildPlanJob(plan, mod, env, !intent.ApplyEnabled(), script, planOutputs[modulePath])
			planJob = &job
			jobs = append(jobs, job)
		}

		if intent.ApplyEnabled() {
			jobs = append(jobs, buildApplyJob(plan, mod, env, planJob, script))
		}
	}

	return jobs
}

func buildPlanJob(plan *jobPlan, mod *discovery.Module, env map[string]string, planOnly bool, script ScriptConfig, outputs PlanOutputs) Job {
	planName := jobName(JobKindPlan, mod)
	modulePath := mod.ID()
	planOperation, produces, artifact := script.NewPlanOperation(planName, modulePath, outputs)

	var deps []JobDependency
	if planOnly {
		deps = controlDependencies(resolveDependencyNames(mod, JobKindPlan, plan.subgraph, plan.moduleIndex))
	} else {
		deps = controlDependencies(resolveDependencyNames(mod, JobKindApply, plan.subgraph, plan.moduleIndex))
	}

	return Job{
		name:           planName,
		kind:           JobKindPlan,
		module:         mod,
		env:            env,
		dependencies:   deps,
		outputArtifact: artifact,
		produces:       produces,
		operation:      planOperation,
	}
}

func buildApplyJob(plan *jobPlan, mod *discovery.Module, env map[string]string, planJob *Job, script ScriptConfig) Job {
	modulePath := mod.ID()
	applyOperation := script.NewApplyOperation(modulePath, planJob != nil)
	applyDeps := controlDependencies(resolveDependencyNames(mod, JobKindApply, plan.subgraph, plan.moduleIndex))
	var consumes []ResourceSpec
	var inputArtifacts []InputArtifact

	if planJob != nil {
		applyDeps = append([]JobDependency{{
			Job: planJob.name,
		}}, applyDeps...)
		consumes = append(consumes,
			PlanResource(ResourceKindPlanBinary, modulePath, applyOperation.terraform.planFile),
		)
		inputArtifacts = append(inputArtifacts, InputArtifact{
			Artifact:    planJob.outputArtifact,
			ProducerJob: planJob.name,
		})
	}

	return Job{
		name:           jobName(JobKindApply, mod),
		kind:           JobKindApply,
		module:         mod,
		env:            env,
		dependencies:   applyDeps,
		inputArtifacts: inputArtifacts,
		consumes:       consumes,
		operation:      applyOperation,
	}
}

func moduleNeedsPlanJob(modulePath string, intent BuildIntent, requests []ResourceRequest) bool {
	if intent.ApplyEnabled() {
		return true
	}
	for _, request := range requests {
		if !isPlanResourceKind(request.kind) {
			continue
		}
		switch request.selector.scope {
		case ResourceScopeAllModules:
			return true
		case ResourceScopeModule:
			if request.selector.modulePath == modulePath {
				return true
			}
		case ResourceScopeAllProducers, ResourceScopeProducer:
			continue
		}
	}
	return false
}

func buildContributedJobs(contributedJobs []ContributedJob) []Job {
	jobs := make([]Job, 0, len(contributedJobs))
	for _, contributedJob := range contributedJobs {
		job := newCommandJob(contributedJob)
		jobs = append(jobs, job)
	}
	return jobs
}

func resolvePipelineResources(ir *IR, plan *jobPlan, required []ResourceRequest, contributedJobs []ContributedJob) error {
	resources, err := buildResourceIndex(ir)
	if err != nil {
		return err
	}

	allowEmptyModuleResources := len(plan.targetModules) == 0
	if _, _, err := resolveResourceRequests(required, resources, "pipeline required resources", allowEmptyModuleResources); err != nil {
		return err
	}

	contributedStart := len(ir.jobs) - len(contributedJobs)
	for i := range contributedJobs {
		job := &ir.jobs[contributedStart+i]
		consumes, artifacts, deps, err := resolveResourceRequestsForJob(contributedJobs[i].Consumes(), resources, job.name, allowEmptyModuleResources)
		if err != nil {
			return err
		}
		job.consumes = consumes
		job.inputArtifacts = artifacts
		for _, dep := range deps {
			job.dependencies = mergeJobDependency(job.dependencies, dep)
		}
	}
	return nil
}
