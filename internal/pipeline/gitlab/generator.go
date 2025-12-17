package gitlab

import (
	"fmt"
	"strings"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/graph"
	"github.com/edelwud/terraci/pkg/config"
)

const (
	// DefaultStagesPrefix is the default prefix for stage names
	DefaultStagesPrefix = "deploy"
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
func (g *Generator) Generate(targetModules []*discovery.Module) (*Pipeline, error) {
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

	// Get effective image (new field or deprecated terraform_image)
	effectiveImage := g.config.GitLab.GetImage()

	pipeline := &Pipeline{
		Stages:    g.generateStages(levels),
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
				pipeline.Jobs[g.jobName(module, "plan")] = planJob
			}

			// Generate apply job (skip if plan-only mode)
			if !g.config.GitLab.PlanOnly {
				applyJob := g.generateApplyJob(module, levelIdx, targetModuleSet)
				pipeline.Jobs[g.jobName(module, "apply")] = applyJob
			}
		}
	}

	return pipeline, nil
}

// generateStages creates stage names for each execution level
func (g *Generator) generateStages(levels [][]string) []string {
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

	return stages
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
	script = append(script, "${TERRAFORM_BINARY} plan -out=plan.tfplan")

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
			Paths:    []string{fmt.Sprintf("%s/plan.tfplan", module.RelativePath)},
			ExpireIn: "1 day",
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
		job.When = "manual"
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

// GenerateForChangedModules generates pipeline only for changed modules and their dependents
func (g *Generator) GenerateForChangedModules(changedModuleIDs []string) (*Pipeline, error) {
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
func (g *Generator) DryRun(targetModules []*discovery.Module) (*DryRunResult, error) {
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

	return &DryRunResult{
		TotalModules:    len(g.modules),
		AffectedModules: len(targetModules),
		Stages:          len(g.generateStages(levels)),
		Jobs:            jobCount,
		ExecutionOrder:  levels,
	}, nil
}
