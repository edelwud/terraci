package gitlabci

import (
	"fmt"
	"maps"
	"strings"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
)

const (
	// DefaultStagesPrefix is the default prefix for stage names
	DefaultStagesPrefix = "deploy"
	// WhenManual is the GitLab CI "when: manual" value for jobs that require manual trigger
	WhenManual = "manual"
)

// Generator generates GitLab CI pipelines
type Generator struct {
	config        *Config
	contributions []*pipeline.Contribution
	depGraph      *graph.DependencyGraph
	modules       []*discovery.Module
	moduleIndex   *discovery.ModuleIndex
}

// NewGenerator creates a new pipeline generator
func NewGenerator(cfg *Config, contributions []*pipeline.Contribution, depGraph *graph.DependencyGraph, modules []*discovery.Module) *Generator {
	return &Generator{
		config:        cfg,
		contributions: contributions,
		depGraph:      depGraph,
		modules:       modules,
		moduleIndex:   discovery.NewModuleIndex(modules),
	}
}

// Generate creates a GitLab CI pipeline for the given modules
func (g *Generator) Generate(targetModules []*discovery.Module) (pipeline.GeneratedPipeline, error) {
	ir, err := g.buildIR(targetModules)
	if err != nil {
		return nil, err
	}

	return g.transform(ir), nil
}

// buildIR constructs the provider-agnostic IR.
func (g *Generator) buildIR(targetModules []*discovery.Module) (*pipeline.IR, error) {
	return pipeline.Build(pipeline.BuildOptions{
		DepGraph:      g.depGraph,
		TargetModules: targetModules,
		AllModules:    g.modules,
		ModuleIndex:   g.moduleIndex,
		Script: pipeline.ScriptConfig{
			TerraformBinary: "${TERRAFORM_BINARY}",
			InitEnabled:     g.config.InitEnabled,
			DetailedPlan:    g.isMREnabled(),
			PlanEnabled:     g.config.PlanEnabled,
			AutoApprove:     g.config.AutoApprove,
		},
		Contributions: g.contributions,
		PlanEnabled:   g.config.PlanEnabled,
		PlanOnly:      g.config.PlanOnly,
	})
}

// transform converts the IR into a GitLab CI Pipeline.
func (g *Generator) transform(ir *pipeline.IR) *Pipeline {
	hasContributed := len(ir.Jobs) > 0

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
		Stages:    g.generateStages(ir, hasContributed),
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

	prefix := g.stagesPrefix()

	// Transform module jobs from each level
	for _, level := range ir.Levels {
		for _, mj := range level.Modules {
			if mj.Plan != nil {
				planJob := g.transformPlanJob(mj.Plan, mj.Module, level.Index, prefix)
				result.Jobs[mj.Plan.Name] = planJob
			}
			if mj.Apply != nil {
				applyJob := g.transformApplyJob(mj.Apply, mj.Module, level.Index, prefix)
				result.Jobs[mj.Apply.Name] = applyJob
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
			result.Jobs[cj.Name] = job
		}
	}

	return result
}

// transformPlanJob converts an IR plan job to a GitLab CI job.
func (g *Generator) transformPlanJob(irJob *pipeline.Job, module *discovery.Module, levelIdx int, prefix string) *Job {
	// Build script with contributed steps injected
	script := g.buildScriptWithSteps(irJob.Script, irJob.Steps, pipeline.PhasePrePlan, pipeline.PhasePostPlan)

	job := &Job{
		Stage:     fmt.Sprintf("%s-plan-%d", prefix, levelIdx),
		Script:    script,
		Variables: irJob.Env,
		Artifacts: &Artifacts{
			Paths:    irJob.ArtifactPaths,
			ExpireIn: "1 day",
			When:     "always",
		},
		Cache:         g.generateCache(module),
		ResourceGroup: module.ID(),
	}

	// Convert dependencies to needs
	job.Needs = toJobNeeds(irJob.Dependencies)

	// Apply job_defaults first, then overwrites
	g.applyJobDefaults(job)
	g.applyOverwrites(job, OverwriteTypePlan)

	return job
}

// transformApplyJob converts an IR apply job to a GitLab CI job.
func (g *Generator) transformApplyJob(irJob *pipeline.Job, module *discovery.Module, levelIdx int, prefix string) *Job {
	// Build script with contributed steps injected
	script := g.buildScriptWithSteps(irJob.Script, irJob.Steps, pipeline.PhasePreApply, pipeline.PhasePostApply)

	job := &Job{
		Stage:         fmt.Sprintf("%s-apply-%d", prefix, levelIdx),
		Script:        script,
		Variables:     irJob.Env,
		Cache:         g.generateCache(module),
		ResourceGroup: module.ID(),
	}

	// Set manual approval if not auto-approve
	if !g.config.AutoApprove {
		job.When = WhenManual
	}

	// Convert dependencies to needs
	job.Needs = toJobNeeds(irJob.Dependencies)

	// Apply job_defaults first, then overwrites
	g.applyJobDefaults(job)
	g.applyOverwrites(job, OverwriteTypeApply)

	return job
}

// transformContributedJob converts an IR contributed job to a GitLab CI job.
func (g *Generator) transformContributedJob(irJob *pipeline.Job) *Job {
	needs := make([]JobNeed, 0, len(irJob.Dependencies))
	for _, dep := range irJob.Dependencies {
		needs = append(needs, JobNeed{Job: dep, Optional: true})
	}

	var script []string
	if irJob.AllowFailure {
		for _, cmd := range irJob.Script {
			script = append(script, cmd+" || true")
		}
	} else {
		script = irJob.Script
	}

	// Look up the stage from the original contributed job data
	stage := g.findContributedJobStage(irJob.Name)

	job := &Job{
		Stage:  stage,
		Script: script,
		Needs:  needs,
	}

	if len(irJob.ArtifactPaths) > 0 {
		job.Artifacts = &Artifacts{
			Paths:    irJob.ArtifactPaths,
			ExpireIn: "1 day",
			When:     "always",
		}
	}

	return job
}

// applySummaryJobOverrides applies MR summary job config overrides (image, tags, rules).
func (g *Generator) applySummaryJobOverrides(job *Job) {
	// Add MR-specific rules
	job.Rules = []Rule{
		{
			If:   "$CI_MERGE_REQUEST_IID",
			When: "always",
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
}

// buildScriptWithSteps injects contributed steps around the core script.
func (g *Generator) buildScriptWithSteps(coreScript []string, steps []pipeline.Step, prePh, postPh pipeline.Phase) []string {
	var script []string
	for _, s := range steps {
		if s.Phase == prePh {
			script = append(script, s.Command)
		}
	}
	script = append(script, coreScript...)
	for _, s := range steps {
		if s.Phase == postPh {
			script = append(script, s.Command)
		}
	}
	return script
}

// generateStages creates stage names for each execution level.
func (g *Generator) generateStages(ir *pipeline.IR, includeContributed bool) []string {
	stages := make([]string, 0)
	prefix := g.stagesPrefix()

	for _, level := range ir.Levels {
		if g.config.PlanEnabled {
			stages = append(stages, fmt.Sprintf("%s-plan-%d", prefix, level.Index))
		}
		if !g.config.PlanOnly {
			stages = append(stages, fmt.Sprintf("%s-apply-%d", prefix, level.Index))
		}
	}

	// Add contributed job stages after the last plan stage
	if includeContributed {
		stages = g.insertContributedStages(stages, prefix, ir)
	}

	return stages
}

// insertContributedStages inserts stages from contributed jobs at appropriate positions.
// Non-finalize stages go after the last plan stage; finalize stages go at the very end.
func (g *Generator) insertContributedStages(stages []string, prefix string, _ *pipeline.IR) []string {
	// Collect unique stages from contributed jobs, separating finalize from others
	seen := make(map[string]bool)
	var contributedStages []string
	var finalizeStages []string
	for _, c := range g.contributions {
		for _, j := range c.Jobs {
			stage := j.Phase.String()
			if !seen[stage] {
				seen[stage] = true
				if j.Phase == pipeline.PhaseFinalize {
					finalizeStages = append(finalizeStages, stage)
				} else {
					contributedStages = append(contributedStages, stage)
				}
			}
		}
	}

	if len(contributedStages) == 0 && len(finalizeStages) == 0 {
		return stages
	}

	// Find the position after the last plan stage
	lastPlanIdx := -1
	for i, stage := range stages {
		if strings.HasPrefix(stage, prefix+"-plan-") {
			lastPlanIdx = i
		}
	}

	if lastPlanIdx == -1 {
		out := make([]string, 0, len(stages)+len(contributedStages)+len(finalizeStages))
		out = append(out, stages...)
		out = append(out, contributedStages...)
		out = append(out, finalizeStages...)
		return out
	}

	// Insert non-finalize contributed stages after the last plan stage
	insertIdx := lastPlanIdx + 1
	result := make([]string, 0, len(stages)+len(contributedStages)+len(finalizeStages))
	result = append(result, stages[:insertIdx]...)
	result = append(result, contributedStages...)
	result = append(result, stages[insertIdx:]...)
	// Finalize stages always go at the very end
	result = append(result, finalizeStages...)
	return result
}

// findContributedJobStage looks up the stage for a contributed job by name.
func (g *Generator) findContributedJobStage(name string) string {
	for _, c := range g.contributions {
		for _, j := range c.Jobs {
			if j.Name == name {
				return j.Phase.String()
			}
		}
	}
	return ""
}

// generateCache creates cache configuration for a module
func (g *Generator) generateCache(module *discovery.Module) *Cache {
	if !g.config.CacheEnabled {
		return nil
	}
	cacheKey := strings.ReplaceAll(module.RelativePath, "/", "-")
	return &Cache{
		Key:   cacheKey,
		Paths: []string{fmt.Sprintf("%s/.terraform/", module.RelativePath)},
	}
}

// applyJobConfig applies job configuration settings to a job
func (g *Generator) applyJobConfig(job *Job, cfg JobConfig) {
	if img := cfg.GetImage(); img != nil && img.Name != "" {
		job.Image = &ImageConfig{
			Name:       img.Name,
			Entrypoint: img.Entrypoint,
		}
	}

	if tokens := cfg.GetIDTokens(); len(tokens) > 0 {
		job.IDTokens = make(map[string]*IDToken)
		for name, token := range tokens {
			job.IDTokens[name] = &IDToken{
				Aud: token.Aud,
			}
		}
	}

	if secrets := cfg.GetSecrets(); len(secrets) > 0 {
		job.Secrets = g.convertSecretsFromOverwrite(secrets)
	}

	if bs := cfg.GetBeforeScript(); len(bs) > 0 {
		job.BeforeScript = bs
	}

	if as := cfg.GetAfterScript(); len(as) > 0 {
		job.AfterScript = as
	}

	if artifacts := cfg.GetArtifacts(); artifacts != nil {
		job.Artifacts = g.convertArtifactsFromOverwrite(artifacts)
	}

	if tags := cfg.GetTags(); len(tags) > 0 {
		job.Tags = tags
	}

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
func (g *Generator) applyOverwrites(job *Job, jobType JobOverwriteType) {
	for i := range g.config.Overwrites {
		ow := &g.config.Overwrites[i]
		if ow.Type != jobType {
			continue
		}
		g.applyJobConfig(job, ow)
	}
}

// convertSecretsFromOverwrite converts overwrite secrets to pipeline secrets
func (g *Generator) convertSecretsFromOverwrite(secrets map[string]CfgSecret) map[string]*Secret {
	result := make(map[string]*Secret)
	for name, secret := range secrets {
		s := &Secret{
			File: secret.File,
		}
		if secret.Vault == nil {
			result[name] = s
			continue
		}
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
		result[name] = s
	}
	return result
}

// convertArtifactsFromOverwrite converts overwrite artifacts to pipeline artifacts
func (g *Generator) convertArtifactsFromOverwrite(cfg *ArtifactsConfig) *Artifacts {
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

// toJobNeeds converts a slice of job name strings to JobNeed structs.
func toJobNeeds(deps []string) []JobNeed {
	if len(deps) == 0 {
		return nil
	}
	needs := make([]JobNeed, len(deps))
	for i, name := range deps {
		needs[i] = JobNeed{Job: name}
	}
	return needs
}

// stagesPrefix returns the configured or default stages prefix.
func (g *Generator) stagesPrefix() string {
	if g.config.StagesPrefix != "" {
		return g.config.StagesPrefix
	}
	return DefaultStagesPrefix
}

// isMREnabled returns true if MR integration is enabled in config
func (g *Generator) isMREnabled() bool {
	if g.config == nil || g.config.MR == nil {
		return false
	}
	if g.config.MR.Comment == nil || g.config.MR.Comment.Enabled == nil {
		return true
	}
	return *g.config.MR.Comment.Enabled
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

// DryRun returns information about what would be generated without creating YAML
func (g *Generator) DryRun(targetModules []*discovery.Module) (*pipeline.DryRunResult, error) {
	ir, err := g.buildIR(targetModules)
	if err != nil {
		return nil, err
	}

	plan, planErr := pipeline.BuildJobPlan(
		g.depGraph, targetModules, g.modules, g.moduleIndex,
		g.hasContributedJobs(), g.config.PlanEnabled,
	)
	if planErr != nil {
		return nil, planErr
	}

	result := pipeline.BuildDryRunResult(plan, len(g.modules), g.config.PlanEnabled)
	// Override stage count with GitLab-specific calculation (plan+apply stages per level)
	result.Stages = len(g.generateStages(ir, len(ir.Jobs) > 0))
	return result, nil
}
