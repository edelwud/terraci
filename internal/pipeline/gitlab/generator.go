// Package gitlab provides GitLab CI pipeline generation
package gitlab

import (
	"fmt"
	"sort"
	"strings"

	"go.yaml.in/yaml/v4"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/graph"
	"github.com/edelwud/terraci/pkg/config"
)

const (
	// DefaultStagesPrefix is the default prefix for stage names
	DefaultStagesPrefix = "deploy"
)

// Pipeline represents a GitLab CI pipeline
type Pipeline struct {
	Stages    []string          `yaml:"stages"`
	Variables map[string]string `yaml:"variables,omitempty"`
	Default   *DefaultConfig    `yaml:"default,omitempty"`
	Jobs      map[string]*Job   `yaml:"-"` // Jobs are added inline
	Workflow  *Workflow         `yaml:"workflow,omitempty"`
}

// DefaultConfig represents default job configuration (only image in default section)
type DefaultConfig struct {
	Image *ImageConfig `yaml:"image,omitempty"`
}

// ImageConfig represents GitLab CI image configuration
// Can be marshaled as either string or object with entrypoint
type ImageConfig struct {
	Name       string   `yaml:"name,omitempty"`
	Entrypoint []string `yaml:"entrypoint,omitempty"`
}

// MarshalYAML implements custom marshaling to output string when no entrypoint
func (img ImageConfig) MarshalYAML() (interface{}, error) {
	if len(img.Entrypoint) == 0 {
		// Simple string format
		return img.Name, nil
	}
	// Object format with entrypoint
	type imageAlias ImageConfig
	return imageAlias(img), nil
}

// IDToken represents GitLab CI OIDC token configuration
type IDToken struct {
	Aud string `yaml:"aud"`
}

// Secret represents GitLab CI secret from external secret manager
type Secret struct {
	Vault     *VaultSecret `yaml:"vault,omitempty"`
	VaultPath string       `yaml:"-"` // For shorthand format
	File      bool         `yaml:"file,omitempty"`
}

// MarshalYAML implements custom marshaling to support vault shorthand format
func (s Secret) MarshalYAML() (interface{}, error) {
	if s.VaultPath != "" {
		// Use shorthand format
		type secretShorthand struct {
			Vault string `yaml:"vault"`
			File  bool   `yaml:"file,omitempty"`
		}
		return secretShorthand{
			Vault: s.VaultPath,
			File:  s.File,
		}, nil
	}
	// Use full format
	type secretAlias Secret
	return secretAlias(s), nil
}

// VaultSecret represents a secret from HashiCorp Vault
// Can be either full object syntax or string shorthand
type VaultSecret struct {
	Engine *VaultEngine `yaml:"engine,omitempty"`
	Path   string       `yaml:"path,omitempty"`
	Field  string       `yaml:"field,omitempty"`
}

// VaultEngine represents Vault secrets engine configuration
type VaultEngine struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

// VaultSecretShorthand is used for string shorthand format in YAML output
type VaultSecretShorthand string

// Job represents a GitLab CI job
type Job struct {
	Stage         string              `yaml:"stage"`
	Image         *ImageConfig        `yaml:"image,omitempty"`
	Script        []string            `yaml:"script"`
	BeforeScript  []string            `yaml:"before_script,omitempty"`
	AfterScript   []string            `yaml:"after_script,omitempty"`
	Variables     map[string]string   `yaml:"variables,omitempty"`
	Needs         []JobNeed           `yaml:"needs,omitempty"`
	Rules         []Rule              `yaml:"rules,omitempty"`
	Artifacts     *Artifacts          `yaml:"artifacts,omitempty"`
	Cache         *Cache              `yaml:"cache,omitempty"`
	Secrets       map[string]*Secret  `yaml:"secrets,omitempty"`
	IDTokens      map[string]*IDToken `yaml:"id_tokens,omitempty"`
	When          string              `yaml:"when,omitempty"`
	AllowFailure  bool                `yaml:"allow_failure,omitempty"`
	Tags          []string            `yaml:"tags,omitempty"`
	ResourceGroup string              `yaml:"resource_group,omitempty"`
}

// Cache represents GitLab CI cache configuration
type Cache struct {
	Key    string   `yaml:"key"`
	Paths  []string `yaml:"paths"`
	Policy string   `yaml:"policy,omitempty"` // pull, push, pull-push
}

// JobNeed represents a job dependency
type JobNeed struct {
	Job      string `yaml:"job"`
	Optional bool   `yaml:"optional,omitempty"`
}

// Rule represents a job rule
type Rule struct {
	If      string   `yaml:"if,omitempty"`
	When    string   `yaml:"when,omitempty"`
	Changes []string `yaml:"changes,omitempty"`
}

// Artifacts represents job artifacts
type Artifacts struct {
	Paths     []string `yaml:"paths,omitempty"`
	ExpireIn  string   `yaml:"expire_in,omitempty"`
	Reports   *Reports `yaml:"reports,omitempty"`
	Name      string   `yaml:"name,omitempty"`
	Untracked bool     `yaml:"untracked,omitempty"`
	When      string   `yaml:"when,omitempty"`
	ExposeAs  string   `yaml:"expose_as,omitempty"`
}

// Reports represents artifact reports
type Reports struct {
	Terraform []string `yaml:"terraform,omitempty"`
	JUnit     []string `yaml:"junit,omitempty"`
	Cobertura []string `yaml:"cobertura,omitempty"`
}

// Workflow controls when pipelines are created
type Workflow struct {
	Rules []Rule `yaml:"rules,omitempty"`
}

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
				pipeline.Jobs[planJob.jobName(module, "plan")] = planJob
			}

			// Generate apply job
			applyJob := g.generateApplyJob(module, levelIdx, targetModuleSet)
			pipeline.Jobs[applyJob.jobName(module, "apply")] = applyJob
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
		stages = append(stages, fmt.Sprintf("%s-apply-%d", prefix, i))
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
	job.Needs = g.getDependencyNeeds(module, "apply", targetModuleSet)

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

// applyJobDefaults applies job_defaults settings to a job
func (g *Generator) applyJobDefaults(job *Job) {
	jd := g.config.GitLab.JobDefaults
	if jd == nil {
		return
	}

	// Apply image
	if jd.Image != nil && jd.Image.Name != "" {
		job.Image = &ImageConfig{
			Name:       jd.Image.Name,
			Entrypoint: jd.Image.Entrypoint,
		}
	}

	// Apply id_tokens
	if len(jd.IDTokens) > 0 {
		job.IDTokens = make(map[string]*IDToken)
		for name, token := range jd.IDTokens {
			job.IDTokens[name] = &IDToken{
				Aud: token.Aud,
			}
		}
	}

	// Apply secrets
	if len(jd.Secrets) > 0 {
		job.Secrets = g.convertSecretsFromOverwrite(jd.Secrets)
	}

	// Apply before_script
	if len(jd.BeforeScript) > 0 {
		job.BeforeScript = jd.BeforeScript
	}

	// Apply after_script
	if len(jd.AfterScript) > 0 {
		job.AfterScript = jd.AfterScript
	}

	// Apply artifacts
	if jd.Artifacts != nil {
		job.Artifacts = g.convertArtifactsFromOverwrite(jd.Artifacts)
	}

	// Apply tags
	if len(jd.Tags) > 0 {
		job.Tags = jd.Tags
	}

	// Apply rules
	if len(jd.Rules) > 0 {
		job.Rules = make([]Rule, len(jd.Rules))
		for i, r := range jd.Rules {
			job.Rules[i] = Rule{
				If:      r.If,
				When:    r.When,
				Changes: r.Changes,
			}
		}
	}

	// Apply variables
	if len(jd.Variables) > 0 {
		if job.Variables == nil {
			job.Variables = make(map[string]string)
		}
		for k, v := range jd.Variables {
			job.Variables[k] = v
		}
	}
}

// applyOverwrites applies job overwrites based on job type
func (g *Generator) applyOverwrites(job *Job, jobType config.JobOverwriteType) {
	for i := range g.config.GitLab.Overwrites {
		ow := &g.config.GitLab.Overwrites[i]
		// Check if this overwrite applies to the job type
		if ow.Type != jobType {
			continue
		}

		// Apply image override
		if ow.Image != nil && ow.Image.Name != "" {
			job.Image = &ImageConfig{
				Name:       ow.Image.Name,
				Entrypoint: ow.Image.Entrypoint,
			}
		}

		// Apply id_tokens override
		if len(ow.IDTokens) > 0 {
			job.IDTokens = make(map[string]*IDToken)
			for name, token := range ow.IDTokens {
				job.IDTokens[name] = &IDToken{
					Aud: token.Aud,
				}
			}
		}

		// Apply secrets override
		if len(ow.Secrets) > 0 {
			job.Secrets = g.convertSecretsFromOverwrite(ow.Secrets)
		}

		// Apply before_script override
		if len(ow.BeforeScript) > 0 {
			job.BeforeScript = ow.BeforeScript
		}

		// Apply after_script override
		if len(ow.AfterScript) > 0 {
			job.AfterScript = ow.AfterScript
		}

		// Apply artifacts override
		if ow.Artifacts != nil {
			job.Artifacts = g.convertArtifactsFromOverwrite(ow.Artifacts)
		}

		// Apply tags override
		if len(ow.Tags) > 0 {
			job.Tags = ow.Tags
		}

		// Apply rules override (job-level)
		if len(ow.Rules) > 0 {
			job.Rules = make([]Rule, len(ow.Rules))
			for i, r := range ow.Rules {
				job.Rules[i] = Rule{
					If:      r.If,
					When:    r.When,
					Changes: r.Changes,
				}
			}
		}

		// Apply variables override
		if len(ow.Variables) > 0 {
			if job.Variables == nil {
				job.Variables = make(map[string]string)
			}
			for k, v := range ow.Variables {
				job.Variables[k] = v
			}
		}
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

// Helper method for Job to generate its name
func (j *Job) jobName(module *discovery.Module, jobType string) string {
	name := strings.ReplaceAll(module.ID(), "/", "-")
	return fmt.Sprintf("%s-%s", jobType, name)
}

// ToYAML converts the pipeline to YAML
func (p *Pipeline) ToYAML() ([]byte, error) {
	// We need custom marshaling to handle jobs properly
	// Create a map that includes all fields

	result := make(map[string]interface{})
	result["stages"] = p.Stages

	if len(p.Variables) > 0 {
		result["variables"] = p.Variables
	}

	if p.Default != nil {
		result["default"] = p.Default
	}

	if p.Workflow != nil {
		result["workflow"] = p.Workflow
	}

	// Add jobs sorted by name
	jobNames := make([]string, 0, len(p.Jobs))
	for name := range p.Jobs {
		jobNames = append(jobNames, name)
	}
	sort.Strings(jobNames)

	for _, name := range jobNames {
		result[name] = p.Jobs[name]
	}

	return yaml.Marshal(result)
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

// GenerateDryRun returns a summary of what would be generated
type DryRunResult struct {
	TotalModules    int
	AffectedModules int
	Stages          int
	Jobs            int
	ExecutionOrder  [][]string
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
