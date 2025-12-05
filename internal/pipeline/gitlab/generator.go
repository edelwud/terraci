// Package gitlab provides GitLab CI pipeline generation
package gitlab

import (
	"fmt"
	"sort"
	"strings"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/graph"
	"github.com/edelwud/terraci/pkg/config"
	"gopkg.in/yaml.v3"
)

// Pipeline represents a GitLab CI pipeline
type Pipeline struct {
	Stages    []string               `yaml:"stages"`
	Variables map[string]string      `yaml:"variables,omitempty"`
	Default   *DefaultConfig         `yaml:"default,omitempty"`
	Jobs      map[string]*Job        `yaml:"-"` // Jobs are added inline
	Workflow  *Workflow              `yaml:"workflow,omitempty"`
}

// DefaultConfig represents default job configuration
type DefaultConfig struct {
	Image        string   `yaml:"image,omitempty"`
	BeforeScript []string `yaml:"before_script,omitempty"`
	AfterScript  []string `yaml:"after_script,omitempty"`
	Tags         []string `yaml:"tags,omitempty"`
}

// Job represents a GitLab CI job
type Job struct {
	Stage        string            `yaml:"stage"`
	Image        string            `yaml:"image,omitempty"`
	Script       []string          `yaml:"script"`
	BeforeScript []string          `yaml:"before_script,omitempty"`
	AfterScript  []string          `yaml:"after_script,omitempty"`
	Variables    map[string]string `yaml:"variables,omitempty"`
	Needs        []JobNeed         `yaml:"needs,omitempty"`
	Rules        []Rule            `yaml:"rules,omitempty"`
	Artifacts    *Artifacts        `yaml:"artifacts,omitempty"`
	When         string            `yaml:"when,omitempty"`
	AllowFailure bool              `yaml:"allow_failure,omitempty"`
	Tags         []string          `yaml:"tags,omitempty"`
	ResourceGroup string           `yaml:"resource_group,omitempty"`
}

// JobNeed represents a job dependency
type JobNeed struct {
	Job      string `yaml:"job"`
	Optional bool   `yaml:"optional,omitempty"`
}

// Rule represents a job rule
type Rule struct {
	If      string `yaml:"if,omitempty"`
	When    string `yaml:"when,omitempty"`
	Changes []string `yaml:"changes,omitempty"`
}

// Artifacts represents job artifacts
type Artifacts struct {
	Paths   []string `yaml:"paths,omitempty"`
	ExpireIn string  `yaml:"expire_in,omitempty"`
	Reports *Reports `yaml:"reports,omitempty"`
}

// Reports represents artifact reports
type Reports struct {
	Terraform []string `yaml:"terraform,omitempty"`
}

// Workflow controls when pipelines are created
type Workflow struct {
	Rules []Rule `yaml:"rules,omitempty"`
}

// Generator generates GitLab CI pipelines
type Generator struct {
	config     *config.Config
	depGraph   *graph.DependencyGraph
	modules    []*discovery.Module
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

	pipeline := &Pipeline{
		Stages:    g.generateStages(levels),
		Variables: variables,
		Default: &DefaultConfig{
			Image:        g.config.GitLab.TerraformImage,
			BeforeScript: g.config.GitLab.BeforeScript,
			AfterScript:  g.config.GitLab.AfterScript,
			Tags:         g.config.GitLab.Tags,
		},
		Jobs: make(map[string]*Job),
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
				planJob := g.generatePlanJob(module, levelIdx, subgraph)
				pipeline.Jobs[planJob.jobName(module, "plan")] = planJob
			}

			// Generate apply job
			applyJob := g.generateApplyJob(module, levelIdx, subgraph)
			pipeline.Jobs[applyJob.jobName(module, "apply")] = applyJob
		}
	}

	return pipeline, nil
}

// generateStages creates stage names for each execution level
func (g *Generator) generateStages(levels [][]string) []string {
	var stages []string
	prefix := g.config.GitLab.StagesPrefix
	if prefix == "" {
		prefix = "deploy"
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
func (g *Generator) generatePlanJob(module *discovery.Module, level int, depGraph *graph.DependencyGraph) *Job {
	prefix := g.config.GitLab.StagesPrefix
	if prefix == "" {
		prefix = "deploy"
	}

	job := &Job{
		Stage: fmt.Sprintf("%s-plan-%d", prefix, level),
		Script: []string{
			fmt.Sprintf("cd %s", module.RelativePath),
			"${TERRAFORM_BINARY} plan -out=plan.tfplan",
		},
		Variables: map[string]string{
			"TF_MODULE_PATH": module.RelativePath,
			"TF_SERVICE":     module.Service,
			"TF_ENVIRONMENT": module.Environment,
			"TF_REGION":      module.Region,
			"TF_MODULE":      module.Name(),
		},
		Artifacts: &Artifacts{
			Paths:    []string{fmt.Sprintf("%s/plan.tfplan", module.RelativePath)},
			ExpireIn: "1 day",
		},
		ResourceGroup: module.ID(),
	}

	// Add needs for dependencies from previous levels
	job.Needs = g.getDependencyNeeds(module, level, depGraph, "apply")

	return job
}

// generateApplyJob creates a terraform apply job
func (g *Generator) generateApplyJob(module *discovery.Module, level int, depGraph *graph.DependencyGraph) *Job {
	prefix := g.config.GitLab.StagesPrefix
	if prefix == "" {
		prefix = "deploy"
	}

	var script []string
	script = append(script, fmt.Sprintf("cd %s", module.RelativePath))

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
		Stage: fmt.Sprintf("%s-apply-%d", prefix, level),
		Script: script,
		Variables: map[string]string{
			"TF_MODULE_PATH": module.RelativePath,
			"TF_SERVICE":     module.Service,
			"TF_ENVIRONMENT": module.Environment,
			"TF_REGION":      module.Region,
			"TF_MODULE":      module.Name(),
		},
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
	depNeeds := g.getDependencyNeeds(module, level, depGraph, "apply")
	needs = append(needs, depNeeds...)

	job.Needs = needs

	return job
}

// getDependencyNeeds returns job needs for a module's dependencies
func (g *Generator) getDependencyNeeds(module *discovery.Module, level int, depGraph *graph.DependencyGraph, jobType string) []JobNeed {
	var needs []JobNeed

	deps := depGraph.GetDependencies(module.ID())
	for _, depID := range deps {
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
