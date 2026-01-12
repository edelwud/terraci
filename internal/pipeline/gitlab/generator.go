package gitlab

import (
	"fmt"
	"strings"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/graph"
	"github.com/edelwud/terraci/internal/pipeline"
	"github.com/edelwud/terraci/pkg/config"
)

const (
	// DefaultStagesPrefix is the default prefix for stage names
	DefaultStagesPrefix = "deploy"
	// SummaryJobName is the name of the summary job
	SummaryJobName = "terraci-summary"
	// SummaryStageName is the name of the summary stage
	SummaryStageName = "summary"
	// PolicyCheckJobName is the name of the policy check job
	PolicyCheckJobName = "policy-check"
	// PolicyCheckStageName is the name of the policy check stage
	PolicyCheckStageName = "policy-check"
	// WhenManual is the GitLab CI "when: manual" value for jobs that require manual trigger
	WhenManual = "manual"
)

// Generator generates GitLab CI pipelines
type Generator struct {
	config      *config.Config
	depGraph    *graph.DependencyGraph
	modules     []*discovery.Module
	moduleIndex *discovery.ModuleIndex
}

// NewGenerator creates a new pipeline generator
func NewGenerator(cfg *config.Config, depGraph *graph.DependencyGraph, modules []*discovery.Module) *Generator {
	return &Generator{
		config:      cfg,
		depGraph:    depGraph,
		modules:     modules,
		moduleIndex: discovery.NewModuleIndex(modules),
	}
}

// Generate creates a GitLab CI pipeline for the given modules
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

	// Merge variables with TERRAFORM_BINARY
	variables := make(map[string]string)
	for k, v := range g.config.GitLab.Variables {
		variables[k] = v
	}
	tfBinary := g.config.GitLab.TerraformBinary
	if tfBinary == "" {
		tfBinary = "terraform"
	}
	variables["TERRAFORM_BINARY"] = tfBinary

	effectiveImage := g.config.GitLab.GetImage()
	includeSummary := g.isMREnabled() && g.config.GitLab.PlanEnabled
	includePolicyCheck := g.isPolicyEnabled() && g.config.GitLab.PlanEnabled

	result := &Pipeline{
		Stages:    g.generateStages(levels, includePolicyCheck, includeSummary),
		Variables: variables,
		Default: &DefaultConfig{
			Image: &ImageConfig{
				Name:       effectiveImage.Name,
				Entrypoint: effectiveImage.Entrypoint,
			},
		},
		Jobs:     make(map[string]*Job),
		Workflow: g.generateWorkflow(),
	}

	// Collect plan job names for summary job dependencies
	var planJobNames []string

	// Generate jobs for each level
	for levelIdx, moduleIDs := range levels {
		for _, moduleID := range moduleIDs {
			module := g.moduleIndex.ByID(moduleID)
			if module == nil {
				continue
			}

			// Generate plan job if enabled
			if g.config.GitLab.PlanEnabled {
				planJob := g.generatePlanJob(module, levelIdx, targetModuleSet)
				planJobName := g.jobName(module, "plan")
				result.Jobs[planJobName] = planJob
				planJobNames = append(planJobNames, planJobName)
			}

			// Generate apply job (skip if plan-only mode)
			if !g.config.GitLab.PlanOnly {
				applyJob := g.generateApplyJob(module, levelIdx, targetModuleSet)
				result.Jobs[g.jobName(module, "apply")] = applyJob
			}
		}
	}

	// Generate policy check job if policy checks are enabled
	if includePolicyCheck && len(planJobNames) > 0 {
		policyJob := g.generatePolicyCheckJob(planJobNames)
		result.Jobs[PolicyCheckJobName] = policyJob
	}

	// Generate summary job if MR integration is enabled
	if includeSummary && len(planJobNames) > 0 {
		summaryJob := g.generateSummaryJob(planJobNames, includePolicyCheck)
		result.Jobs[SummaryJobName] = summaryJob
	}

	return result, nil
}

// generateStages creates stage names for each execution level
func (g *Generator) generateStages(levels [][]string, includePolicyCheck, includeSummary bool) []string {
	stages := make([]string, 0)
	prefix := g.config.GitLab.StagesPrefix
	if prefix == "" {
		prefix = DefaultStagesPrefix
	}

	for i := range levels {
		if g.config.GitLab.PlanEnabled {
			stages = append(stages, fmt.Sprintf("%s-plan-%d", prefix, i))
		}
		if !g.config.GitLab.PlanOnly {
			stages = append(stages, fmt.Sprintf("%s-apply-%d", prefix, i))
		}
	}

	// Add policy-check stage after all plans but before applies
	// When plan-only mode, add after all plans
	if includePolicyCheck {
		stages = insertPolicyCheckStage(stages, prefix)
	}

	// Add summary stage if MR integration is enabled
	if includeSummary {
		stages = append(stages, SummaryStageName)
	}

	return stages
}

// insertPolicyCheckStage inserts the policy-check stage after the last plan stage
func insertPolicyCheckStage(stages []string, prefix string) []string {
	// Find the position after the last plan stage
	lastPlanIdx := -1
	for i, stage := range stages {
		if strings.HasPrefix(stage, prefix+"-plan-") {
			lastPlanIdx = i
		}
	}

	if lastPlanIdx == -1 {
		// No plan stages, append at end
		return append(stages, PolicyCheckStageName)
	}

	// Insert after the last plan stage
	insertIdx := lastPlanIdx + 1
	result := make([]string, 0, len(stages)+1)
	result = append(result, stages[:insertIdx]...)
	result = append(result, PolicyCheckStageName)
	result = append(result, stages[insertIdx:]...)
	return result
}

// generatePlanJob creates a terraform plan job
func (g *Generator) generatePlanJob(module *discovery.Module, level int, targetModuleSet map[string]bool) *Job {
	prefix := g.config.GitLab.StagesPrefix
	if prefix == "" {
		prefix = DefaultStagesPrefix
	}

	// Build script with cd, optional init, and plan
	script := []string{fmt.Sprintf("cd %s", module.RelativePath)}
	if g.config.GitLab.InitEnabled {
		script = append(script, "${TERRAFORM_BINARY} init")
	}

	// Determine artifacts paths
	artifactsPaths := []string{fmt.Sprintf("%s/plan.tfplan", module.RelativePath)}

	// If MR integration is enabled, capture plan output for summary
	if g.isMREnabled() {
		// Use -detailed-exitcode and capture output to plan.txt
		// Save exit code to file, generate JSON plan, then exit with saved code
		// Exit code 2 = changes present (success), 1 = error, 0 = no changes
		script = append(script,
			"(${TERRAFORM_BINARY} plan -out=plan.tfplan -detailed-exitcode 2>&1 || echo $? > .tf_exit) | tee plan.txt",
			// Generate JSON plan for detailed parsing by summary job
			"${TERRAFORM_BINARY} show -json plan.tfplan > plan.json",
			// Exit with appropriate code: 2 (changes) -> 0, others pass through
			"TF_EXIT=$(cat .tf_exit 2>/dev/null || echo 0); rm -f .tf_exit; if [ \"$TF_EXIT\" -eq 2 ]; then exit 0; else exit \"$TF_EXIT\"; fi")
		// Add plan.txt and plan.json to artifacts for summary job
		artifactsPaths = append(artifactsPaths,
			fmt.Sprintf("%s/plan.txt", module.RelativePath),
			fmt.Sprintf("%s/plan.json", module.RelativePath))
	} else {
		script = append(script, "${TERRAFORM_BINARY} plan -out=plan.tfplan")
	}

	job := &Job{
		Stage:  fmt.Sprintf("%s-plan-%d", prefix, level),
		Script: script,
		Variables: map[string]string{
			"TF_MODULE_PATH": module.RelativePath,
			"TF_SERVICE":     module.Service,
			"TF_ENVIRONMENT": module.Environment,
			"TF_REGION":      module.Region,
			"TF_MODULE":      module.Name(),
		},
		// Default artifacts for plan - can be overridden via job_defaults or overwrites
		Artifacts: &Artifacts{
			Paths:    artifactsPaths,
			ExpireIn: "1 day",
			When:     "always", // Save artifacts even on failure for summary
		},
		Cache:         g.generateCache(module),
		ResourceGroup: module.ID(),
	}

	// Add needs for dependencies from previous levels
	// In plan-only mode, depend on plan jobs; otherwise depend on apply jobs
	if g.config.GitLab.PlanOnly {
		job.Needs = g.getDependencyNeeds(module, "plan", targetModuleSet)
	} else {
		job.Needs = g.getDependencyNeeds(module, "apply", targetModuleSet)
	}

	// Apply job_defaults first, then overwrites
	g.applyJobDefaults(job)
	g.applyOverwrites(job, config.OverwriteTypePlan)

	return job
}

// generateApplyJob creates a terraform apply job
func (g *Generator) generateApplyJob(module *discovery.Module, level int, targetModuleSet map[string]bool) *Job {
	prefix := g.config.GitLab.StagesPrefix
	if prefix == "" {
		prefix = DefaultStagesPrefix
	}

	// Build script with cd, optional init, and apply
	script := []string{fmt.Sprintf("cd %s", module.RelativePath)}
	if g.config.GitLab.InitEnabled {
		script = append(script, "${TERRAFORM_BINARY} init")
	}

	if g.config.GitLab.PlanEnabled {
		script = append(script, "${TERRAFORM_BINARY} apply plan.tfplan")
	} else {
		if g.config.GitLab.AutoApprove {
			script = append(script, "${TERRAFORM_BINARY} apply -auto-approve")
		} else {
			script = append(script, "${TERRAFORM_BINARY} apply")
		}
	}

	job := &Job{
		Stage:  fmt.Sprintf("%s-apply-%d", prefix, level),
		Script: script,
		Variables: map[string]string{
			"TF_MODULE_PATH": module.RelativePath,
			"TF_SERVICE":     module.Service,
			"TF_ENVIRONMENT": module.Environment,
			"TF_REGION":      module.Region,
			"TF_MODULE":      module.Name(),
		},
		Cache:         g.generateCache(module),
		ResourceGroup: module.ID(),
	}

	// Set manual approval if not auto-approve
	if !g.config.GitLab.AutoApprove {
		job.When = WhenManual
	}

	// Add needs
	var needs []JobNeed

	// Need the plan job for this module
	if g.config.GitLab.PlanEnabled {
		needs = append(needs, JobNeed{
			Job: g.jobName(module, "plan"),
		})
	}

	// Need apply jobs from dependencies
	depNeeds := g.getDependencyNeeds(module, "apply", targetModuleSet)
	needs = append(needs, depNeeds...)

	job.Needs = needs

	// Apply job_defaults first, then overwrites
	g.applyJobDefaults(job)
	g.applyOverwrites(job, config.OverwriteTypeApply)

	return job
}

// generateCache creates cache configuration for a module
func (g *Generator) generateCache(module *discovery.Module) *Cache {
	// Return nil if caching is disabled
	if !g.config.GitLab.CacheEnabled {
		return nil
	}

	// Convert module path to cache key (replace slashes with dashes)
	cacheKey := strings.ReplaceAll(module.RelativePath, "/", "-")

	return &Cache{
		Key:   cacheKey,
		Paths: []string{fmt.Sprintf("%s/.terraform/", module.RelativePath)},
	}
}

// applyJobConfig applies job configuration settings to a job
func (g *Generator) applyJobConfig(job *Job, cfg config.JobConfig) {
	// Apply image
	if img := cfg.GetImage(); img != nil && img.Name != "" {
		job.Image = &ImageConfig{
			Name:       img.Name,
			Entrypoint: img.Entrypoint,
		}
	}

	// Apply id_tokens
	if tokens := cfg.GetIDTokens(); len(tokens) > 0 {
		job.IDTokens = make(map[string]*IDToken)
		for name, token := range tokens {
			job.IDTokens[name] = &IDToken{
				Aud: token.Aud,
			}
		}
	}

	// Apply secrets
	if secrets := cfg.GetSecrets(); len(secrets) > 0 {
		job.Secrets = g.convertSecretsFromOverwrite(secrets)
	}

	// Apply before_script
	if bs := cfg.GetBeforeScript(); len(bs) > 0 {
		job.BeforeScript = bs
	}

	// Apply after_script
	if as := cfg.GetAfterScript(); len(as) > 0 {
		job.AfterScript = as
	}

	// Apply artifacts
	if artifacts := cfg.GetArtifacts(); artifacts != nil {
		job.Artifacts = g.convertArtifactsFromOverwrite(artifacts)
	}

	// Apply tags
	if tags := cfg.GetTags(); len(tags) > 0 {
		job.Tags = tags
	}

	// Apply rules
	if rules := cfg.GetRules(); len(rules) > 0 {
		job.Rules = make([]Rule, len(rules))
		for i, r := range rules {
			job.Rules[i] = Rule{
				If:      r.If,
				When:    r.When,
				Changes: r.Changes,
			}
		}
	}

	// Apply variables
	if vars := cfg.GetVariables(); len(vars) > 0 {
		if job.Variables == nil {
			job.Variables = make(map[string]string)
		}
		for k, v := range vars {
			job.Variables[k] = v
		}
	}
}

// applyJobDefaults applies job_defaults settings to a job
func (g *Generator) applyJobDefaults(job *Job) {
	if g.config.GitLab.JobDefaults == nil {
		return
	}
	g.applyJobConfig(job, g.config.GitLab.JobDefaults)
}

// applyOverwrites applies job overwrites based on job type
func (g *Generator) applyOverwrites(job *Job, jobType config.JobOverwriteType) {
	for i := range g.config.GitLab.Overwrites {
		ow := &g.config.GitLab.Overwrites[i]
		if ow.Type != jobType {
			continue
		}
		g.applyJobConfig(job, ow)
	}
}

// convertSecretsFromOverwrite converts overwrite secrets to pipeline secrets
func (g *Generator) convertSecretsFromOverwrite(secrets map[string]config.Secret) map[string]*Secret {
	result := make(map[string]*Secret)
	for name, secret := range secrets {
		s := &Secret{
			File: secret.File,
		}
		if secret.Vault != nil {
			if secret.Vault.Shorthand != "" {
				s.VaultPath = secret.Vault.Shorthand
			} else {
				s.Vault = &VaultSecret{
					Path:  secret.Vault.Path,
					Field: secret.Vault.Field,
				}
				if secret.Vault.Engine != nil {
					s.Vault.Engine = &VaultEngine{
						Name: secret.Vault.Engine.Name,
						Path: secret.Vault.Engine.Path,
					}
				}
			}
		}
		result[name] = s
	}
	return result
}

// convertArtifactsFromOverwrite converts overwrite artifacts to pipeline artifacts
func (g *Generator) convertArtifactsFromOverwrite(cfg *config.ArtifactsConfig) *Artifacts {
	if cfg == nil {
		return nil
	}
	artifacts := &Artifacts{
		Paths:     cfg.Paths,
		ExpireIn:  cfg.ExpireIn,
		Name:      cfg.Name,
		Untracked: cfg.Untracked,
		When:      cfg.When,
		ExposeAs:  cfg.ExposeAs,
	}
	if cfg.Reports != nil {
		artifacts.Reports = &Reports{
			Terraform: cfg.Reports.Terraform,
			JUnit:     cfg.Reports.JUnit,
			Cobertura: cfg.Reports.Cobertura,
		}
	}
	return artifacts
}

// generateWorkflow creates workflow configuration with rules
func (g *Generator) generateWorkflow() *Workflow {
	if len(g.config.GitLab.Rules) == 0 {
		return nil
	}

	rules := make([]Rule, len(g.config.GitLab.Rules))
	for i, r := range g.config.GitLab.Rules {
		rules[i] = Rule{
			If:      r.If,
			When:    r.When,
			Changes: r.Changes,
		}
	}

	return &Workflow{
		Rules: rules,
	}
}

// getDependencyNeeds returns job needs for a module's dependencies
// Only includes dependencies that are in the targetModuleSet (i.e., have jobs generated)
func (g *Generator) getDependencyNeeds(module *discovery.Module, jobType string, targetModuleSet map[string]bool) []JobNeed {
	needs := make([]JobNeed, 0)

	deps := g.depGraph.GetDependencies(module.ID())
	for _, depID := range deps {
		// Skip dependencies that are not in the target set (no job generated for them)
		if !targetModuleSet[depID] {
			continue
		}

		depModule := g.moduleIndex.ByID(depID)
		if depModule == nil {
			continue
		}

		needs = append(needs, JobNeed{
			Job: g.jobName(depModule, jobType),
		})
	}

	return needs
}

// jobName generates a job name for a module
func (g *Generator) jobName(module *discovery.Module, jobType string) string {
	// Create a safe job name from module path
	name := strings.ReplaceAll(module.ID(), "/", "-")
	return fmt.Sprintf("%s-%s", jobType, name)
}

// isMREnabled returns true if MR integration is enabled in config
func (g *Generator) isMREnabled() bool {
	if g.config.GitLab.MR == nil {
		return false
	}
	if g.config.GitLab.MR.Comment == nil {
		return true // Default enabled when MR section exists
	}
	if g.config.GitLab.MR.Comment.Enabled == nil {
		return true // Default enabled
	}
	return *g.config.GitLab.MR.Comment.Enabled
}

// generateSummaryJob creates the terraci summary job that posts MR comments
func (g *Generator) generateSummaryJob(planJobNames []string, includePolicyCheck bool) *Job {
	// Build needs from all plan jobs (with artifacts)
	needs := make([]JobNeed, 0, len(planJobNames)+1)
	for _, jobName := range planJobNames {
		needs = append(needs, JobNeed{Job: jobName, Optional: true})
	}

	// If policy check is enabled, also depend on it
	if includePolicyCheck {
		needs = append(needs, JobNeed{Job: PolicyCheckJobName, Optional: true})
	}

	job := &Job{
		Stage:  SummaryStageName,
		Script: []string{"terraci summary"},
		Needs:  needs,
		Rules: []Rule{
			{
				If:   "$CI_MERGE_REQUEST_IID",
				When: "always",
			},
		},
	}

	// Apply summary job configuration if specified
	if g.config.GitLab.MR != nil && g.config.GitLab.MR.SummaryJob != nil {
		sjCfg := g.config.GitLab.MR.SummaryJob
		if sjCfg.Image != nil && sjCfg.Image.Name != "" {
			job.Image = &ImageConfig{
				Name:       sjCfg.Image.Name,
				Entrypoint: sjCfg.Image.Entrypoint,
			}
		}
		if len(sjCfg.Tags) > 0 {
			job.Tags = sjCfg.Tags
		}
	}

	return job
}

// isPolicyEnabled returns true if policy checks are enabled in config
func (g *Generator) isPolicyEnabled() bool {
	return g.config.Policy != nil && g.config.Policy.Enabled
}

// generatePolicyCheckJob creates the policy check job
func (g *Generator) generatePolicyCheckJob(planJobNames []string) *Job {
	// Build needs from all plan jobs (with artifacts)
	needs := make([]JobNeed, len(planJobNames))
	for i, jobName := range planJobNames {
		needs[i] = JobNeed{Job: jobName, Optional: true}
	}

	// Determine exit behavior based on on_failure setting
	var script []string
	if g.config.Policy.OnFailure == config.PolicyActionWarn {
		// Don't fail the job on policy violations, just warn
		script = []string{
			"terraci policy pull",
			"terraci policy check || true",
		}
	} else {
		// Block on policy violations (default)
		script = []string{
			"terraci policy pull",
			"terraci policy check",
		}
	}

	job := &Job{
		Stage:  PolicyCheckStageName,
		Script: script,
		Needs:  needs,
		Artifacts: &Artifacts{
			Paths:    []string{".terraci/policy-results.json"},
			ExpireIn: "1 day",
			When:     "always",
		},
	}

	// Use the same image as summary job if specified
	if g.config.GitLab.MR != nil && g.config.GitLab.MR.SummaryJob != nil {
		sjCfg := g.config.GitLab.MR.SummaryJob
		if sjCfg.Image != nil && sjCfg.Image.Name != "" {
			job.Image = &ImageConfig{
				Name:       sjCfg.Image.Name,
				Entrypoint: sjCfg.Image.Entrypoint,
			}
		}
		if len(sjCfg.Tags) > 0 {
			job.Tags = sjCfg.Tags
		}
	}

	return job
}

// GenerateForChangedModules generates pipeline only for changed modules and their dependents
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

	jobCount := 0
	for _, level := range levels {
		jobCount += len(level)
		if g.config.GitLab.PlanEnabled {
			jobCount += len(level) // plan + apply
		}
	}

	includeSummary := g.isMREnabled() && g.config.GitLab.PlanEnabled
	includePolicyCheck := g.isPolicyEnabled() && g.config.GitLab.PlanEnabled

	if includePolicyCheck {
		jobCount++ // Add policy check job
	}
	if includeSummary {
		jobCount++ // Add summary job
	}

	return &pipeline.DryRunResult{
		TotalModules:    len(g.modules),
		AffectedModules: len(targetModules),
		Stages:          len(g.generateStages(levels, includePolicyCheck, includeSummary)),
		Jobs:            jobCount,
		ExecutionOrder:  levels,
	}, nil
}
