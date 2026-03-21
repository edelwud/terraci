package gitlab

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
	config      *config.GitLabConfig
	policyCfg   *config.PolicyConfig
	depGraph    *graph.DependencyGraph
	modules     []*discovery.Module
	moduleIndex *discovery.ModuleIndex
}

// NewGenerator creates a new pipeline generator
func NewGenerator(cfg *config.GitLabConfig, policyCfg *config.PolicyConfig, depGraph *graph.DependencyGraph, modules []*discovery.Module) *Generator {
	return &Generator{
		config:      cfg,
		policyCfg:   policyCfg,
		depGraph:    depGraph,
		modules:     modules,
		moduleIndex: discovery.NewModuleIndex(modules),
	}
}

// Generate creates a GitLab CI pipeline for the given modules
func (g *Generator) Generate(targetModules []*discovery.Module) (pipeline.GeneratedPipeline, error) {
	plan, err := pipeline.BuildJobPlan(
		g.depGraph, targetModules, g.modules, g.moduleIndex,
		g.isMREnabled(), g.isPolicyEnabled(), g.config.PlanEnabled,
	)
	if err != nil {
		return nil, err
	}

	// Merge variables with TERRAFORM_BINARY
	variables := make(map[string]string)
	maps.Copy(variables, g.config.Variables)
	tfBinary := g.config.TerraformBinary
	if tfBinary == "" {
		tfBinary = "terraform"
	}
	variables["TERRAFORM_BINARY"] = tfBinary

	effectiveImage := g.config.GetImage()

	result := &Pipeline{
		Stages:    g.generateStages(plan.ExecutionLevels, plan.IncludePolicy, plan.IncludeSummary),
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
	for levelIdx, moduleIDs := range plan.ExecutionLevels {
		for _, moduleID := range moduleIDs {
			module := plan.ModuleIndex.ByID(moduleID)
			if module == nil {
				continue
			}

			// Generate plan job if enabled
			if g.config.PlanEnabled {
				planJob := g.generatePlanJob(module, levelIdx, plan.TargetSet)
				planJobName := g.jobName(module, "plan")
				result.Jobs[planJobName] = planJob
				planJobNames = append(planJobNames, planJobName)
			}

			// Generate apply job (skip if plan-only mode)
			if !g.config.PlanOnly {
				applyJob := g.generateApplyJob(module, levelIdx, plan.TargetSet)
				result.Jobs[g.jobName(module, "apply")] = applyJob
			}
		}
	}

	// Generate policy check job if policy checks are enabled
	if plan.IncludePolicy && len(planJobNames) > 0 {
		policyJob := g.generatePolicyCheckJob(planJobNames)
		result.Jobs[PolicyCheckJobName] = policyJob
	}

	// Generate summary job if MR integration is enabled
	if plan.IncludeSummary && len(planJobNames) > 0 {
		summaryJob := g.generateSummaryJob(planJobNames, plan.IncludePolicy)
		result.Jobs[SummaryJobName] = summaryJob
	}

	return result, nil
}

// generateStages creates stage names for each execution level
func (g *Generator) generateStages(levels [][]string, includePolicyCheck, includeSummary bool) []string {
	stages := make([]string, 0)
	prefix := g.config.StagesPrefix
	if prefix == "" {
		prefix = DefaultStagesPrefix
	}

	for i := range levels {
		if g.config.PlanEnabled {
			stages = append(stages, fmt.Sprintf("%s-plan-%d", prefix, i))
		}
		if !g.config.PlanOnly {
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
	prefix := g.config.StagesPrefix
	if prefix == "" {
		prefix = DefaultStagesPrefix
	}

	// Build script and artifact paths using shared ScriptConfig
	sc := pipeline.ScriptConfig{
		TerraformBinary: "${TERRAFORM_BINARY}",
		InitEnabled:     g.config.InitEnabled,
		DetailedPlan:    g.isMREnabled(),
	}
	script, artifactsPaths := sc.PlanScript(module.RelativePath)

	job := &Job{
		Stage:     fmt.Sprintf("%s-plan-%d", prefix, level),
		Script:    script,
		Variables: pipeline.BuildModuleEnvVars(module),
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
	if g.config.PlanOnly {
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
	prefix := g.config.StagesPrefix
	if prefix == "" {
		prefix = DefaultStagesPrefix
	}

	// Build script using shared ScriptConfig
	sc := pipeline.ScriptConfig{
		TerraformBinary: "${TERRAFORM_BINARY}",
		InitEnabled:     g.config.InitEnabled,
		PlanEnabled:     g.config.PlanEnabled,
		AutoApprove:     g.config.AutoApprove,
	}
	script := sc.ApplyScript(module.RelativePath)

	job := &Job{
		Stage:         fmt.Sprintf("%s-apply-%d", prefix, level),
		Script:        script,
		Variables:     pipeline.BuildModuleEnvVars(module),
		Cache:         g.generateCache(module),
		ResourceGroup: module.ID(),
	}

	// Set manual approval if not auto-approve
	if !g.config.AutoApprove {
		job.When = WhenManual
	}

	// Add needs
	var needs []JobNeed

	// Need the plan job for this module
	if g.config.PlanEnabled {
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
	if !g.config.CacheEnabled {
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
		maps.Copy(job.Variables, vars)
	}
}

// applyJobDefaults applies job_defaults settings to a job
func (g *Generator) applyJobDefaults(job *Job) {
	if g.config.JobDefaults == nil {
		return
	}
	g.applyJobConfig(job, g.config.JobDefaults)
}

// applyOverwrites applies job overwrites based on job type
func (g *Generator) applyOverwrites(job *Job, jobType config.JobOverwriteType) {
	for i := range g.config.Overwrites {
		ow := &g.config.Overwrites[i]
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
	if len(g.config.Rules) == 0 {
		return nil
	}

	rules := make([]Rule, len(g.config.Rules))
	for i, r := range g.config.Rules {
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
	names := pipeline.ResolveDependencyNames(module, jobType, targetModuleSet, g.depGraph, g.moduleIndex)
	needs := make([]JobNeed, len(names))
	for i, name := range names {
		needs[i] = JobNeed{Job: name}
	}
	return needs
}

// jobName generates a job name for a module
func (g *Generator) jobName(module *discovery.Module, jobType string) string {
	return pipeline.JobName(jobType, module)
}

// isMREnabled returns true if MR integration is enabled in config
func (g *Generator) isMREnabled() bool {
	if g.config.MR == nil {
		return false
	}
	if g.config.MR.Comment == nil {
		return true // Default enabled when MR section exists
	}
	if g.config.MR.Comment.Enabled == nil {
		return true // Default enabled
	}
	return *g.config.MR.Comment.Enabled
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
	if g.config.MR != nil && g.config.MR.SummaryJob != nil {
		sjCfg := g.config.MR.SummaryJob
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
	return g.policyCfg != nil && g.policyCfg.Enabled
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
	if g.policyCfg.OnFailure == config.PolicyActionWarn {
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
	if g.config.MR != nil && g.config.MR.SummaryJob != nil {
		sjCfg := g.config.MR.SummaryJob
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

// DryRun returns information about what would be generated without creating YAML
func (g *Generator) DryRun(targetModules []*discovery.Module) (*pipeline.DryRunResult, error) {
	plan, err := pipeline.BuildJobPlan(
		g.depGraph, targetModules, g.modules, g.moduleIndex,
		g.isMREnabled(), g.isPolicyEnabled(), g.config.PlanEnabled,
	)
	if err != nil {
		return nil, err
	}

	result := pipeline.BuildDryRunResult(plan, len(g.modules), g.config.PlanEnabled)
	// Override stage count with GitLab-specific calculation (plan+apply stages per level)
	result.Stages = len(g.generateStages(plan.ExecutionLevels, plan.IncludePolicy, plan.IncludeSummary))
	return result, nil
}
