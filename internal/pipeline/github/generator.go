package github

import (
	"fmt"
	"maps"
	"strings"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/graph"
	"github.com/edelwud/terraci/internal/pipeline"
	"github.com/edelwud/terraci/pkg/config"
)

const (
	// SummaryJobName is the name of the summary job
	SummaryJobName = "terraci-summary"
	// PolicyCheckJobName is the name of the policy check job
	PolicyCheckJobName = "policy-check"
	// stepsInitialCap is the initial capacity for job steps slice
	stepsInitialCap = 8
)

// Generator generates GitHub Actions workflows
type Generator struct {
	config      *config.Config
	depGraph    *graph.DependencyGraph
	modules     []*discovery.Module
	moduleIndex *discovery.ModuleIndex
}

// NewGenerator creates a new GitHub Actions pipeline generator
func NewGenerator(cfg *config.Config, depGraph *graph.DependencyGraph, modules []*discovery.Module) *Generator {
	return &Generator{
		config:      cfg,
		depGraph:    depGraph,
		modules:     modules,
		moduleIndex: discovery.NewModuleIndex(modules),
	}
}

// Generate creates a GitHub Actions workflow for the given modules
func (g *Generator) Generate(targetModules []*discovery.Module) (pipeline.GeneratedPipeline, error) {
	if len(targetModules) == 0 {
		targetModules = g.modules
	}

	// Get module IDs for subgraph
	moduleIDs := make([]string, len(targetModules))
	for i, m := range targetModules {
		moduleIDs[i] = m.ID()
	}

	// Build set of target module IDs for filtering needs
	targetModuleSet := make(map[string]bool, len(moduleIDs))
	for _, id := range moduleIDs {
		targetModuleSet[id] = true
	}

	// Build subgraph for target modules
	subgraph := g.depGraph.Subgraph(moduleIDs)

	// Get execution levels
	levels, err := subgraph.ExecutionLevels()
	if err != nil {
		return nil, fmt.Errorf("failed to calculate execution levels: %w", err)
	}

	ghCfg := g.ghConfig()

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

	includeSummary := g.isPREnabled() && ghCfg.PlanEnabled
	includePolicyCheck := g.isPolicyEnabled() && ghCfg.PlanEnabled

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
	for _, moduleIDs := range levels {
		for _, moduleID := range moduleIDs {
			module := g.moduleIndex.ByID(moduleID)
			if module == nil {
				continue
			}

			// Generate plan job if enabled
			if ghCfg.PlanEnabled {
				planJob := g.generatePlanJob(module, targetModuleSet)
				planJobName := g.jobName(module, "plan")
				workflow.Jobs[planJobName] = planJob
				planJobNames = append(planJobNames, planJobName)
			}

			// Generate apply job (skip if plan-only mode)
			if !ghCfg.PlanOnly {
				applyJob := g.generateApplyJob(module, targetModuleSet)
				workflow.Jobs[g.jobName(module, "apply")] = applyJob
			}
		}
	}

	// Generate policy check job if policy checks are enabled
	if includePolicyCheck && len(planJobNames) > 0 {
		workflow.Jobs[PolicyCheckJobName] = g.generatePolicyCheckJob(planJobNames)
	}

	// Generate summary job if PR integration is enabled
	if includeSummary && len(planJobNames) > 0 {
		workflow.Jobs[SummaryJobName] = g.generateSummaryJob(planJobNames, includePolicyCheck)
	}

	return workflow, nil
}

// generatePlanJob creates a terraform plan job
func (g *Generator) generatePlanJob(module *discovery.Module, targetModuleSet map[string]bool) *Job {
	ghCfg := g.ghConfig()

	// Build run script
	var scriptParts []string
	scriptParts = append(scriptParts, fmt.Sprintf("cd %s", module.RelativePath))
	if ghCfg.InitEnabled {
		scriptParts = append(scriptParts, "${TERRAFORM_BINARY} init")
	}

	if g.isPREnabled() {
		scriptParts = append(scriptParts,
			"(${TERRAFORM_BINARY} plan -out=plan.tfplan -detailed-exitcode 2>&1 || echo $? > .tf_exit) | tee plan.txt",
			"${TERRAFORM_BINARY} show -json plan.tfplan > plan.json",
			"TF_EXIT=$(cat .tf_exit 2>/dev/null || echo 0); rm -f .tf_exit; if [ \"$TF_EXIT\" -eq 2 ]; then exit 0; else exit \"$TF_EXIT\"; fi",
		)
	} else {
		scriptParts = append(scriptParts, "${TERRAFORM_BINARY} plan -out=plan.tfplan")
	}
	runScript := strings.Join(scriptParts, "\n")

	// Build steps (preallocate with capacity for checkout + before/after steps + plan + upload)
	steps := make([]Step, 0, stepsInitialCap)
	steps = append(steps, Step{Name: "Checkout", Uses: "actions/checkout@v4"})

	// Add steps_before from job_defaults
	steps = append(steps, g.getStepsBefore(config.OverwriteTypePlan)...)

	steps = append(steps, Step{
		Name: fmt.Sprintf("Plan %s", module.ID()),
		Run:  runScript,
	})

	// Add steps_after from job_defaults
	steps = append(steps, g.getStepsAfter(config.OverwriteTypePlan)...)

	// Upload artifact step
	artifactPaths := fmt.Sprintf("%s/plan.tfplan", module.RelativePath)
	if g.isPREnabled() {
		artifactPaths = fmt.Sprintf("%s/plan.tfplan\n%s/plan.txt\n%s/plan.json",
			module.RelativePath, module.RelativePath, module.RelativePath)
	}
	steps = append(steps, Step{
		Name: "Upload plan artifacts",
		Uses: "actions/upload-artifact@v4",
		With: map[string]string{
			"name":           g.jobName(module, "plan"),
			"path":           artifactPaths,
			"retention-days": "1",
		},
		If: "always()",
	})

	job := &Job{
		RunsOn: g.getRunsOn(),
		Env: map[string]string{
			"TF_MODULE_PATH": module.RelativePath,
			"TF_SERVICE":     module.Service,
			"TF_ENVIRONMENT": module.Environment,
			"TF_REGION":      module.Region,
			"TF_MODULE":      module.Name(),
		},
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
		job.Needs = g.getDependencyNeeds(module, "plan", targetModuleSet)
	} else {
		job.Needs = g.getDependencyNeeds(module, "apply", targetModuleSet)
	}

	return job
}

// generateApplyJob creates a terraform apply job
func (g *Generator) generateApplyJob(module *discovery.Module, targetModuleSet map[string]bool) *Job {
	ghCfg := g.ghConfig()

	// Build run script
	var scriptParts []string
	scriptParts = append(scriptParts, fmt.Sprintf("cd %s", module.RelativePath))
	if ghCfg.InitEnabled {
		scriptParts = append(scriptParts, "${TERRAFORM_BINARY} init")
	}

	if ghCfg.PlanEnabled {
		scriptParts = append(scriptParts, "${TERRAFORM_BINARY} apply plan.tfplan")
	} else {
		if ghCfg.AutoApprove {
			scriptParts = append(scriptParts, "${TERRAFORM_BINARY} apply -auto-approve")
		} else {
			scriptParts = append(scriptParts, "${TERRAFORM_BINARY} apply")
		}
	}
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
				"name": g.jobName(module, "plan"),
			},
		})
	}

	// Add steps_before from job_defaults
	steps = append(steps, g.getStepsBefore(config.OverwriteTypeApply)...)

	steps = append(steps, Step{
		Name: fmt.Sprintf("Apply %s", module.ID()),
		Run:  runScript,
	})

	// Add steps_after from job_defaults
	steps = append(steps, g.getStepsAfter(config.OverwriteTypeApply)...)

	job := &Job{
		RunsOn: g.getRunsOn(),
		Env: map[string]string{
			"TF_MODULE_PATH": module.RelativePath,
			"TF_SERVICE":     module.Service,
			"TF_ENVIRONMENT": module.Environment,
			"TF_REGION":      module.Region,
			"TF_MODULE":      module.Name(),
		},
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
		needs = append(needs, g.jobName(module, "plan"))
	}
	depNeeds := g.getDependencyNeeds(module, "apply", targetModuleSet)
	needs = append(needs, depNeeds...)
	job.Needs = needs

	return job
}

// generateSummaryJob creates the terraci summary job that posts PR comments
func (g *Generator) generateSummaryJob(planJobNames []string, includePolicyCheck bool) *Job {
	needs := make([]string, 0, len(planJobNames)+1)
	needs = append(needs, planJobNames...)
	if includePolicyCheck {
		needs = append(needs, PolicyCheckJobName)
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

// generatePolicyCheckJob creates the policy check job
func (g *Generator) generatePolicyCheckJob(planJobNames []string) *Job {
	needs := make([]string, len(planJobNames))
	copy(needs, planJobNames)

	// Determine script based on on_failure setting
	var policyScript string
	if g.config.Policy.OnFailure == config.PolicyActionWarn {
		policyScript = "terraci policy pull\nterraci policy check || true"
	} else {
		policyScript = "terraci policy pull\nterraci policy check"
	}

	steps := []Step{
		{Name: "Checkout", Uses: "actions/checkout@v4"},
		{
			Name: "Download all plan artifacts",
			Uses: "actions/download-artifact@v4",
		},
		{
			Name: "Run policy checks",
			Run:  policyScript,
		},
		{
			Name: "Upload policy results",
			Uses: "actions/upload-artifact@v4",
			With: map[string]string{
				"name":           "policy-results",
				"path":           ".terraci/policy-results.json",
				"retention-days": "1",
			},
			If: "always()",
		},
	}

	job := &Job{
		RunsOn: g.getRunsOn(),
		Needs:  needs,
		Steps:  steps,
	}

	// Use summary job runner if specified
	if g.ghConfig().PR != nil && g.ghConfig().PR.SummaryJob != nil {
		if runsOn := g.ghConfig().PR.SummaryJob.RunsOn; runsOn != "" {
			job.RunsOn = runsOn
		}
	}

	return job
}

// GenerateForChangedModules generates workflow only for changed modules and their dependents
func (g *Generator) GenerateForChangedModules(changedModuleIDs []string) (pipeline.GeneratedPipeline, error) {
	// Get all affected modules (changed + their dependents)
	affectedIDs := g.depGraph.GetAffectedModules(changedModuleIDs)

	// Convert to modules
	var affectedModules []*discovery.Module
	for _, id := range affectedIDs {
		if m := g.moduleIndex.ByID(id); m != nil {
			affectedModules = append(affectedModules, m)
		}
	}

	return g.Generate(affectedModules)
}

// DryRun returns information about what would be generated without creating YAML
func (g *Generator) DryRun(targetModules []*discovery.Module) (*pipeline.DryRunResult, error) {
	if len(targetModules) == 0 {
		targetModules = g.modules
	}

	moduleIDs := make([]string, len(targetModules))
	for i, m := range targetModules {
		moduleIDs[i] = m.ID()
	}

	subgraph := g.depGraph.Subgraph(moduleIDs)
	levels, err := subgraph.ExecutionLevels()
	if err != nil {
		return nil, err
	}

	ghCfg := g.ghConfig()

	jobCount := 0
	for _, level := range levels {
		jobCount += len(level)
		if ghCfg.PlanEnabled {
			jobCount += len(level) // plan + apply
		}
	}

	includeSummary := g.isPREnabled() && ghCfg.PlanEnabled
	includePolicyCheck := g.isPolicyEnabled() && ghCfg.PlanEnabled

	if includePolicyCheck {
		jobCount++
	}
	if includeSummary {
		jobCount++
	}

	// Stages in GitHub Actions are conceptual (levels), count them
	stageCount := len(levels)
	if includePolicyCheck {
		stageCount++
	}
	if includeSummary {
		stageCount++
	}

	return &pipeline.DryRunResult{
		TotalModules:    len(g.modules),
		AffectedModules: len(targetModules),
		Stages:          stageCount,
		Jobs:            jobCount,
		ExecutionOrder:  levels,
	}, nil
}

// jobName generates a job name for a module
func (g *Generator) jobName(module *discovery.Module, jobType string) string {
	name := strings.ReplaceAll(module.ID(), "/", "-")
	return fmt.Sprintf("%s-%s", jobType, name)
}

// getDependencyNeeds returns job needs for a module's dependencies
func (g *Generator) getDependencyNeeds(module *discovery.Module, jobType string, targetModuleSet map[string]bool) []string {
	var needs []string

	deps := g.depGraph.GetDependencies(module.ID())
	for _, depID := range deps {
		if !targetModuleSet[depID] {
			continue
		}

		depModule := g.moduleIndex.ByID(depID)
		if depModule == nil {
			continue
		}

		needs = append(needs, g.jobName(depModule, jobType))
	}

	return needs
}

// ghConfig returns the GitHub config with defaults
func (g *Generator) ghConfig() *config.GitHubConfig {
	if g.config.GitHub == nil {
		return &config.GitHubConfig{
			TerraformBinary: "terraform",
			RunsOn:          "ubuntu-latest",
			PlanEnabled:     true,
			InitEnabled:     true,
		}
	}
	return g.config.GitHub
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

// isPolicyEnabled returns true if policy checks are enabled
func (g *Generator) isPolicyEnabled() bool {
	return g.config.Policy != nil && g.config.Policy.Enabled
}

// getStepsBefore returns extra steps to insert before terraform commands
func (g *Generator) getStepsBefore(jobType config.JobOverwriteType) []Step {
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
func (g *Generator) getStepsAfter(jobType config.JobOverwriteType) []Step {
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

// convertConfigStep converts a config.GitHubStep to a workflow Step
func convertConfigStep(s config.GitHubStep) Step {
	return Step{
		Name: s.Name,
		Uses: s.Uses,
		With: s.With,
		Run:  s.Run,
		Env:  s.Env,
	}
}
