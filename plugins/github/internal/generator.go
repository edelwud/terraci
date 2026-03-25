package githubci

import (
	"fmt"
	"maps"
	"strings"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
)

const (
	// SummaryJobName is the name of the summary job
	SummaryJobName = "terraci-summary"
	// stepsInitialCap is the initial capacity for job steps slice
	stepsInitialCap = 8
)

// Generator generates GitHub Actions workflows
type Generator struct {
	config      *Config
	steps       []plugin.PipelineStep
	jobs        []plugin.PipelineJob
	depGraph    *graph.DependencyGraph
	modules     []*discovery.Module
	moduleIndex *discovery.ModuleIndex
}

// NewGenerator creates a new GitHub Actions pipeline generator
func NewGenerator(cfg *Config, steps []plugin.PipelineStep, jobs []plugin.PipelineJob, depGraph *graph.DependencyGraph, modules []*discovery.Module) *Generator {
	return &Generator{
		config:      cfg,
		steps:       steps,
		jobs:        jobs,
		depGraph:    depGraph,
		modules:     modules,
		moduleIndex: discovery.NewModuleIndex(modules),
	}
}

// Generate creates a GitHub Actions workflow for the given modules
func (g *Generator) Generate(targetModules []*discovery.Module) (pipeline.GeneratedPipeline, error) {
	ghCfg := g.ghConfig()

	plan, err := pipeline.BuildJobPlan(
		g.depGraph, targetModules, g.modules, g.moduleIndex,
		g.isPREnabled(), g.hasContributedJobs(), ghCfg.PlanEnabled,
	)
	if err != nil {
		return nil, err
	}

	// Build workflow-level env
	env := make(map[string]string)
	maps.Copy(env, ghCfg.Env)
	tfBinary := ghCfg.TerraformBinary
	if tfBinary == "" {
		tfBinary = "terraform"
	}
	env["TERRAFORM_BINARY"] = tfBinary

	// Build permissions
	permissions := ghCfg.Permissions
	if len(permissions) == 0 {
		permissions = map[string]string{
			"contents":      "read",
			"pull-requests": "write",
		}
	}

	workflow := &Workflow{
		Name: "Terraform",
		On: WorkflowTrigger{
			Push:        &PushTrigger{Branches: []string{"main"}},
			PullRequest: &PRTrigger{Branches: []string{"main"}},
		},
		Permissions: permissions,
		Env:         env,
		Jobs:        make(map[string]*Job),
	}

	// Collect plan job names for summary/policy dependencies
	var planJobNames []string

	// Generate jobs for each level
	for _, moduleIDs := range plan.ExecutionLevels {
		for _, moduleID := range moduleIDs {
			module := plan.ModuleIndex.ByID(moduleID)
			if module == nil {
				continue
			}

			// Generate plan job if enabled
			if ghCfg.PlanEnabled {
				planJob := g.generatePlanJob(module, plan.TargetSet)
				planJobName := pipeline.JobName("plan", module)
				workflow.Jobs[planJobName] = planJob
				planJobNames = append(planJobNames, planJobName)
			}

			// Generate apply job (skip if plan-only mode)
			if !ghCfg.PlanOnly {
				applyJob := g.generateApplyJob(module, plan.TargetSet)
				workflow.Jobs[pipeline.JobName("apply", module)] = applyJob
			}
		}
	}

	// Generate contributed jobs (e.g., policy-check) from PipelineContributor plugins
	if plan.IncludePolicy && len(planJobNames) > 0 {
		contributedJobs := g.generateContributedJobs(planJobNames)
		for name, job := range contributedJobs {
			workflow.Jobs[name] = job
		}
	}

	// Generate summary job if PR integration is enabled
	if plan.IncludeSummary && len(planJobNames) > 0 {
		workflow.Jobs[SummaryJobName] = g.generateSummaryJob(planJobNames, plan.IncludePolicy)
	}

	return workflow, nil
}

// generatePlanJob creates a terraform plan job
func (g *Generator) generatePlanJob(module *discovery.Module, targetModuleSet map[string]bool) *Job {
	ghCfg := g.ghConfig()

	// Build run script
	sc := pipeline.ScriptConfig{
		TerraformBinary: ghCfg.TerraformBinary,
		InitEnabled:     ghCfg.InitEnabled,
		PlanEnabled:     ghCfg.PlanEnabled,
		AutoApprove:     ghCfg.AutoApprove,
		DetailedPlan:    g.isPREnabled(),
	}
	scriptParts, artifactPaths := sc.PlanScript(module.RelativePath)
	runScript := strings.Join(scriptParts, "\n")

	// Build steps (preallocate with capacity for checkout + before/after steps + plan + upload)
	steps := make([]Step, 0, stepsInitialCap)
	steps = append(steps, Step{Name: "Checkout", Uses: "actions/checkout@v4"})

	// Add steps_before from job_defaults
	steps = append(steps, g.getStepsBefore(OverwriteTypePlan)...)

	// Add contributed pre-plan steps
	for _, s := range g.steps {
		if s.Phase == plugin.PhasePrePlan {
			steps = append(steps, Step{Name: s.Name, Run: s.Command})
		}
	}

	steps = append(steps, Step{
		Name: fmt.Sprintf("Plan %s", module.ID()),
		Run:  runScript,
	})

	// Add contributed post-plan steps
	for _, s := range g.steps {
		if s.Phase == plugin.PhasePostPlan {
			steps = append(steps, Step{Name: s.Name, Run: s.Command})
		}
	}

	// Add steps_after from job_defaults
	steps = append(steps, g.getStepsAfter(OverwriteTypePlan)...)

	// Upload artifact step
	steps = append(steps, Step{
		Name: "Upload plan artifacts",
		Uses: "actions/upload-artifact@v4",
		With: map[string]string{
			"name":           pipeline.JobName("plan", module),
			"path":           strings.Join(artifactPaths, "\n"),
			"retention-days": "1",
		},
		If: "always()",
	})

	job := &Job{
		RunsOn: g.getRunsOn(),
		Env:    pipeline.BuildModuleEnvVars(module),
		Concurrency: &Concurrency{
			Group:            module.ID(),
			CancelInProgress: false,
		},
		Steps: steps,
	}

	// Add container if configured
	if container := g.getContainer(); container != nil {
		job.Container = container
	}

	// Add needs for dependencies from previous levels
	if ghCfg.PlanOnly {
		job.Needs = pipeline.ResolveDependencyNames(module, "plan", targetModuleSet, g.depGraph, g.moduleIndex)
	} else {
		job.Needs = pipeline.ResolveDependencyNames(module, "apply", targetModuleSet, g.depGraph, g.moduleIndex)
	}

	return job
}

// generateApplyJob creates a terraform apply job
func (g *Generator) generateApplyJob(module *discovery.Module, targetModuleSet map[string]bool) *Job {
	ghCfg := g.ghConfig()

	// Build run script
	sc := pipeline.ScriptConfig{
		TerraformBinary: ghCfg.TerraformBinary,
		InitEnabled:     ghCfg.InitEnabled,
		PlanEnabled:     ghCfg.PlanEnabled,
		AutoApprove:     ghCfg.AutoApprove,
		DetailedPlan:    g.isPREnabled(),
	}
	scriptParts := sc.ApplyScript(module.RelativePath)
	runScript := strings.Join(scriptParts, "\n")

	// Build steps
	steps := []Step{
		{Name: "Checkout", Uses: "actions/checkout@v4"},
	}

	// Download plan artifact if plan is enabled
	if ghCfg.PlanEnabled {
		steps = append(steps, Step{
			Name: "Download plan artifacts",
			Uses: "actions/download-artifact@v4",
			With: map[string]string{
				"name": pipeline.JobName("plan", module),
			},
		})
	}

	// Add steps_before from job_defaults
	steps = append(steps, g.getStepsBefore(OverwriteTypeApply)...)

	// Add contributed pre-apply steps
	for _, s := range g.steps {
		if s.Phase == plugin.PhasePreApply {
			steps = append(steps, Step{Name: s.Name, Run: s.Command})
		}
	}

	steps = append(steps, Step{
		Name: fmt.Sprintf("Apply %s", module.ID()),
		Run:  runScript,
	})

	// Add contributed post-apply steps
	for _, s := range g.steps {
		if s.Phase == plugin.PhasePostApply {
			steps = append(steps, Step{Name: s.Name, Run: s.Command})
		}
	}

	// Add steps_after from job_defaults
	steps = append(steps, g.getStepsAfter(OverwriteTypeApply)...)

	job := &Job{
		RunsOn: g.getRunsOn(),
		Env:    pipeline.BuildModuleEnvVars(module),
		Concurrency: &Concurrency{
			Group:            module.ID(),
			CancelInProgress: false,
		},
		Steps: steps,
	}

	// Add container if configured
	if container := g.getContainer(); container != nil {
		job.Container = container
	}

	// Set environment for manual approval gate
	if !ghCfg.AutoApprove {
		job.Environment = "production"
	}

	// Add needs
	var needs []string
	if ghCfg.PlanEnabled {
		needs = append(needs, pipeline.JobName("plan", module))
	}
	depNeeds := pipeline.ResolveDependencyNames(module, "apply", targetModuleSet, g.depGraph, g.moduleIndex)
	needs = append(needs, depNeeds...)
	job.Needs = needs

	return job
}

// generateSummaryJob creates the terraci summary job that posts PR comments
func (g *Generator) generateSummaryJob(planJobNames []string, includeContributed bool) *Job {
	needs := make([]string, 0, len(planJobNames)+len(g.jobs)+1)
	needs = append(needs, planJobNames...)
	if includeContributed {
		for _, j := range g.jobs {
			needs = append(needs, j.Name)
		}
	}

	steps := []Step{
		{Name: "Checkout", Uses: "actions/checkout@v4"},
		{
			Name: "Download all plan artifacts",
			Uses: "actions/download-artifact@v4",
		},
		{
			Name: "Post summary",
			Run:  "terraci summary",
		},
	}

	job := &Job{
		RunsOn: g.getRunsOn(),
		Needs:  needs,
		If:     "github.event_name == 'pull_request'",
		Steps:  steps,
	}

	// Apply summary job runner override
	if g.ghConfig().PR != nil && g.ghConfig().PR.SummaryJob != nil {
		if runsOn := g.ghConfig().PR.SummaryJob.RunsOn; runsOn != "" {
			job.RunsOn = runsOn
		}
	}

	return job
}

// generateContributedJobs creates jobs from PipelineContributor plugins.
func (g *Generator) generateContributedJobs(planJobNames []string) map[string]*Job {
	result := make(map[string]*Job)
	for _, j := range g.jobs {
		needs := make([]string, 0)
		if j.DependsOnPlan {
			needs = append(needs, planJobNames...)
		}

		var scriptLines []string
		if j.AllowFailure {
			for _, cmd := range j.Commands {
				scriptLines = append(scriptLines, cmd+" || true")
			}
		} else {
			scriptLines = j.Commands
		}

		jobSteps := []Step{
			{Name: "Checkout", Uses: "actions/checkout@v4"},
			{
				Name: "Download all plan artifacts",
				Uses: "actions/download-artifact@v4",
			},
			{
				Name: fmt.Sprintf("Run %s", j.Name),
				Run:  strings.Join(scriptLines, "\n"),
			},
		}

		// Add artifact upload step if artifact paths specified
		if len(j.ArtifactPaths) > 0 {
			jobSteps = append(jobSteps, Step{
				Name: fmt.Sprintf("Upload %s results", j.Name),
				Uses: "actions/upload-artifact@v4",
				With: map[string]string{
					"name":           j.Name + "-results",
					"path":           strings.Join(j.ArtifactPaths, "\n"),
					"retention-days": "1",
				},
				If: "always()",
			})
		}

		job := &Job{
			RunsOn: g.getRunsOn(),
			Needs:  needs,
			Steps:  jobSteps,
		}

		// Use summary job runner if specified
		if g.ghConfig().PR != nil && g.ghConfig().PR.SummaryJob != nil {
			if runsOn := g.ghConfig().PR.SummaryJob.RunsOn; runsOn != "" {
				job.RunsOn = runsOn
			}
		}

		result[j.Name] = job
	}
	return result
}

// DryRun returns information about what would be generated without creating YAML
func (g *Generator) DryRun(targetModules []*discovery.Module) (*pipeline.DryRunResult, error) {
	ghCfg := g.ghConfig()

	plan, err := pipeline.BuildJobPlan(
		g.depGraph, targetModules, g.modules, g.moduleIndex,
		g.isPREnabled(), g.hasContributedJobs(), ghCfg.PlanEnabled,
	)
	if err != nil {
		return nil, err
	}

	return pipeline.BuildDryRunResult(plan, len(g.modules), ghCfg.PlanEnabled), nil
}

// ghConfig returns the GitHub config with defaults
func (g *Generator) ghConfig() *Config {
	if g.config == nil {
		return &Config{
			TerraformBinary: "terraform",
			RunsOn:          "ubuntu-latest",
			PlanEnabled:     true,
			InitEnabled:     true,
		}
	}
	return g.config
}

// getRunsOn returns the runner label from config
func (g *Generator) getRunsOn() string {
	runsOn := g.ghConfig().RunsOn
	if runsOn == "" {
		return "ubuntu-latest"
	}
	return runsOn
}

// getContainer returns a container config if configured
func (g *Generator) getContainer() *Container {
	ghCfg := g.ghConfig()

	// Check job_defaults container first
	if ghCfg.JobDefaults != nil && ghCfg.JobDefaults.Container != nil {
		return &Container{
			Image: ghCfg.JobDefaults.Container.Name,
		}
	}

	// Fall back to top-level container
	if ghCfg.Container != nil {
		return &Container{
			Image: ghCfg.Container.Name,
		}
	}

	return nil
}

// isPREnabled returns true if PR integration is enabled
func (g *Generator) isPREnabled() bool {
	ghCfg := g.ghConfig()
	if ghCfg.PR == nil {
		return false
	}
	if ghCfg.PR.Comment == nil {
		return true // Default enabled when PR section exists
	}
	if ghCfg.PR.Comment.Enabled == nil {
		return true
	}
	return *ghCfg.PR.Comment.Enabled
}

// hasContributedJobs returns true if any PipelineContributor plugins contributed jobs.
func (g *Generator) hasContributedJobs() bool { return len(g.jobs) > 0 }

// getStepsBefore returns extra steps to insert before terraform commands
func (g *Generator) getStepsBefore(jobType JobOverwriteType) []Step {
	var steps []Step

	// From job_defaults
	ghCfg := g.ghConfig()
	if ghCfg.JobDefaults != nil {
		for _, s := range ghCfg.JobDefaults.StepsBefore {
			steps = append(steps, convertConfigStep(s))
		}
	}

	// From overwrites matching job type
	for _, ow := range ghCfg.Overwrites {
		if ow.Type != jobType {
			continue
		}
		for _, s := range ow.StepsBefore {
			steps = append(steps, convertConfigStep(s))
		}
	}

	return steps
}

// getStepsAfter returns extra steps to insert after terraform commands
func (g *Generator) getStepsAfter(jobType JobOverwriteType) []Step {
	var steps []Step

	ghCfg := g.ghConfig()
	if ghCfg.JobDefaults != nil {
		for _, s := range ghCfg.JobDefaults.StepsAfter {
			steps = append(steps, convertConfigStep(s))
		}
	}

	for _, ow := range ghCfg.Overwrites {
		if ow.Type != jobType {
			continue
		}
		for _, s := range ow.StepsAfter {
			steps = append(steps, convertConfigStep(s))
		}
	}

	return steps
}

// convertConfigStep converts a ConfigStep to a workflow Step
func convertConfigStep(s ConfigStep) Step {
	return Step{
		Name: s.Name,
		Uses: s.Uses,
		With: s.With,
		Run:  s.Run,
		Env:  s.Env,
	}
}
