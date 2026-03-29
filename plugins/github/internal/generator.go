package githubci

import (
	"fmt"
	"maps"
	"strings"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
)

const (
	// stepsInitialCap is the initial capacity for job steps slice
	stepsInitialCap = 8
)

// Generator generates GitHub Actions workflows
type Generator struct {
	config        *Config
	contributions []*pipeline.Contribution
	depGraph      *graph.DependencyGraph
	modules       []*discovery.Module
	moduleIndex   *discovery.ModuleIndex
}

// NewGenerator creates a new GitHub Actions pipeline generator
func NewGenerator(cfg *Config, contributions []*pipeline.Contribution, depGraph *graph.DependencyGraph, modules []*discovery.Module) *Generator {
	return &Generator{
		config:        cfg,
		contributions: contributions,
		depGraph:      depGraph,
		modules:       modules,
		moduleIndex:   discovery.NewModuleIndex(modules),
	}
}

// Generate creates a GitHub Actions workflow for the given modules
func (g *Generator) Generate(targetModules []*discovery.Module) (pipeline.GeneratedPipeline, error) {
	ir, err := g.buildIR(targetModules)
	if err != nil {
		return nil, err
	}

	return g.transform(ir), nil
}

// buildIR constructs the provider-agnostic IR.
func (g *Generator) buildIR(targetModules []*discovery.Module) (*pipeline.IR, error) {
	ghCfg := g.ghConfig()
	return pipeline.Build(pipeline.BuildOptions{
		DepGraph:      g.depGraph,
		TargetModules: targetModules,
		AllModules:    g.modules,
		ModuleIndex:   g.moduleIndex,
		Script: pipeline.ScriptConfig{
			TerraformBinary: ghCfg.TerraformBinary,
			InitEnabled:     ghCfg.InitEnabled,
			PlanEnabled:     ghCfg.PlanEnabled,
			AutoApprove:     ghCfg.AutoApprove,
			DetailedPlan:    g.isPREnabled(),
		},
		Contributions: g.contributions,
		PlanEnabled:   ghCfg.PlanEnabled,
		PlanOnly:      ghCfg.PlanOnly,
	})
}

// transform converts the IR into a GitHub Actions Workflow.
func (g *Generator) transform(ir *pipeline.IR) *Workflow {
	ghCfg := g.ghConfig()
	hasContributed := len(ir.Jobs) > 0

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

	// Transform module jobs
	for _, level := range ir.Levels {
		for _, mj := range level.Modules {
			if mj.Plan != nil {
				workflow.Jobs[mj.Plan.Name] = g.transformPlanJob(mj.Plan, mj.Module)
			}
			if mj.Apply != nil {
				workflow.Jobs[mj.Apply.Name] = g.transformApplyJob(mj.Apply, mj.Module)
			}
		}
	}

	// Transform contributed jobs (including summary if provided by plugin)
	if hasContributed {
		for i := range ir.Jobs {
			cj := &ir.Jobs[i]
			job := g.transformContributedJob(cj)
			// Apply summary job overrides for finalize-phase jobs
			if cj.Phase == pipeline.PhaseFinalize {
				g.applySummaryJobOverrides(job)
			}
			workflow.Jobs[cj.Name] = job
		}
	}

	return workflow
}

// transformPlanJob converts an IR plan job to a GitHub Actions job.
func (g *Generator) transformPlanJob(irJob *pipeline.Job, module *discovery.Module) *Job {
	ghCfg := g.ghConfig()

	runScript := strings.Join(irJob.Script, "\n")
	artifactPaths := irJob.ArtifactPaths

	// Build steps
	steps := make([]Step, 0, stepsInitialCap)
	steps = append(steps, Step{Name: "Checkout", Uses: "actions/checkout@v4"})

	// Add steps_before from job_defaults
	steps = append(steps, g.getStepsBefore(OverwriteTypePlan)...)

	// Add contributed pre-plan steps
	for _, s := range irJob.Steps {
		if s.Phase == pipeline.PhasePrePlan {
			steps = append(steps, Step{Name: s.Name, Run: s.Command})
		}
	}

	steps = append(steps, Step{
		Name: "Plan " + module.ID(),
		Run:  runScript,
	})

	// Add contributed post-plan steps
	for _, s := range irJob.Steps {
		if s.Phase == pipeline.PhasePostPlan {
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
			"name":           irJob.Name,
			"path":           strings.Join(artifactPaths, "\n"),
			"retention-days": "1",
		},
		If: "always()",
	})

	job := &Job{
		RunsOn: g.getRunsOn(),
		Env:    irJob.Env,
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

	// Dependencies
	if ghCfg.PlanOnly {
		job.Needs = pipeline.ResolveDependencyNames(module, "plan", g.targetSet(irJob), g.depGraph, g.moduleIndex)
	} else {
		job.Needs = pipeline.ResolveDependencyNames(module, "apply", g.targetSet(irJob), g.depGraph, g.moduleIndex)
	}

	return job
}

// transformApplyJob converts an IR apply job to a GitHub Actions job.
func (g *Generator) transformApplyJob(irJob *pipeline.Job, module *discovery.Module) *Job {
	ghCfg := g.ghConfig()

	runScript := strings.Join(irJob.Script, "\n")

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
	for _, s := range irJob.Steps {
		if s.Phase == pipeline.PhasePreApply {
			steps = append(steps, Step{Name: s.Name, Run: s.Command})
		}
	}

	steps = append(steps, Step{
		Name: "Apply " + module.ID(),
		Run:  runScript,
	})

	// Add contributed post-apply steps
	for _, s := range irJob.Steps {
		if s.Phase == pipeline.PhasePostApply {
			steps = append(steps, Step{Name: s.Name, Run: s.Command})
		}
	}

	// Add steps_after from job_defaults
	steps = append(steps, g.getStepsAfter(OverwriteTypeApply)...)

	job := &Job{
		RunsOn: g.getRunsOn(),
		Env:    irJob.Env,
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

	// Dependencies come from IR
	job.Needs = irJob.Dependencies

	return job
}

// transformContributedJob converts an IR contributed job to a GitHub Actions job.
func (g *Generator) transformContributedJob(irJob *pipeline.Job) *Job {
	var scriptLines []string
	if irJob.AllowFailure {
		for _, cmd := range irJob.Script {
			scriptLines = append(scriptLines, cmd+" || true")
		}
	} else {
		scriptLines = irJob.Script
	}

	jobSteps := []Step{
		{Name: "Checkout", Uses: "actions/checkout@v4"},
		{
			Name: "Download all plan artifacts",
			Uses: "actions/download-artifact@v4",
		},
		{
			Name: "Run " + irJob.Name,
			Run:  strings.Join(scriptLines, "\n"),
		},
	}

	// Add artifact upload step if artifact paths specified
	if len(irJob.ArtifactPaths) > 0 {
		jobSteps = append(jobSteps, Step{
			Name: fmt.Sprintf("Upload %s results", irJob.Name),
			Uses: "actions/upload-artifact@v4",
			With: map[string]string{
				"name":           irJob.Name + "-results",
				"path":           strings.Join(irJob.ArtifactPaths, "\n"),
				"retention-days": "1",
			},
			If: "always()",
		})
	}

	job := &Job{
		RunsOn: g.getRunsOn(),
		Needs:  irJob.Dependencies,
		Steps:  jobSteps,
	}

	return job
}

// applySummaryJobOverrides applies PR summary job config overrides.
func (g *Generator) applySummaryJobOverrides(job *Job) {
	// Add PR-specific condition
	job.If = "github.event_name == 'pull_request'"

	// Apply summary job runner override
	if g.ghConfig().PR != nil && g.ghConfig().PR.SummaryJob != nil {
		if runsOn := g.ghConfig().PR.SummaryJob.RunsOn; runsOn != "" {
			job.RunsOn = runsOn
		}
	}
}

// targetSet builds a target set by scanning the IR's module index.
// This is a convenience method that reconstructs the target set from the dep graph.
func (g *Generator) targetSet(_ *pipeline.Job) map[string]bool {
	// Build target set from all modules known to the generator
	targetSet := make(map[string]bool, len(g.modules))
	for _, m := range g.modules {
		targetSet[m.ID()] = true
	}
	return targetSet
}

// DryRun returns information about what would be generated without creating YAML
func (g *Generator) DryRun(targetModules []*discovery.Module) (*pipeline.DryRunResult, error) {
	ghCfg := g.ghConfig()

	plan, err := pipeline.BuildJobPlan(
		g.depGraph, targetModules, g.modules, g.moduleIndex,
		g.hasContributedJobs(), ghCfg.PlanEnabled,
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

	if ghCfg.JobDefaults != nil && ghCfg.JobDefaults.Container != nil {
		return &Container{
			Image: ghCfg.JobDefaults.Container.Name,
		}
	}

	if ghCfg.Container != nil {
		return &Container{
			Image: ghCfg.Container.Name,
		}
	}

	return nil
}

// isPREnabled returns true if PR integration is enabled
func (g *Generator) isPREnabled() bool {
	if g.config == nil || g.config.PR == nil {
		return false
	}
	if g.config.PR.Comment == nil || g.config.PR.Comment.Enabled == nil {
		return true
	}
	return *g.config.PR.Comment.Enabled
}

// hasContributedJobs returns true if any contributions have jobs.
func (g *Generator) hasContributedJobs() bool {
	for _, c := range g.contributions {
		if len(c.Jobs) > 0 {
			return true
		}
	}
	return false
}

// getStepsBefore returns extra steps to insert before terraform commands
func (g *Generator) getStepsBefore(jobType JobOverwriteType) []Step {
	var steps []Step

	ghCfg := g.ghConfig()
	if ghCfg.JobDefaults != nil {
		for _, s := range ghCfg.JobDefaults.StepsBefore {
			steps = append(steps, convertConfigStep(s))
		}
	}

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
