package domain

import (
	"errors"
	"fmt"
	"maps"
	"sort"

	"go.yaml.in/yaml/v4"
)

// Pipeline represents a GitLab CI pipeline.
type Pipeline struct {
	stages    []string
	variables map[string]string
	defaults  *DefaultConfig
	jobs      map[string]Job
	workflow  *Workflow
}

type PipelineOptions struct {
	Stages    []string
	Variables map[string]string
	Default   *DefaultConfig
	Workflow  *Workflow
	Jobs      []NamedJob
}

type NamedJob struct {
	Name string
	Job  Job
}

type PipelineBuilder struct {
	opts PipelineOptions
	jobs map[string]Job
}

func NewPipeline(opts PipelineOptions) (*Pipeline, error) {
	builder := NewPipelineBuilder(PipelineOptions{
		Stages:    opts.Stages,
		Variables: opts.Variables,
		Default:   opts.Default,
		Workflow:  opts.Workflow,
	})
	for i := range opts.Jobs {
		job := opts.Jobs[i]
		if err := builder.AddJob(job.Name, job.Job); err != nil {
			return nil, err
		}
	}
	return builder.Build()
}

func NewPipelineBuilder(opts PipelineOptions) *PipelineBuilder {
	return &PipelineBuilder{
		opts: PipelineOptions{
			Stages:    append([]string(nil), opts.Stages...),
			Variables: cloneStringMap(opts.Variables),
			Default:   cloneDefaultConfig(opts.Default),
			Workflow:  cloneWorkflow(opts.Workflow),
		},
		jobs: make(map[string]Job),
	}
}

func (b *PipelineBuilder) AddJob(name string, job Job) error {
	if b == nil {
		return errors.New("gitlab pipeline builder is nil")
	}
	if name == "" {
		return errors.New("gitlab job name is required")
	}
	if _, exists := b.jobs[name]; exists {
		return fmt.Errorf("duplicate gitlab job %q", name)
	}
	b.jobs[name] = job.clone()
	return nil
}

func (b *PipelineBuilder) Build() (*Pipeline, error) {
	if b == nil {
		return nil, errors.New("gitlab pipeline builder is nil")
	}
	jobs := make([]NamedJob, 0, len(b.jobs))
	for name := range b.jobs {
		job := b.jobs[name]
		jobs = append(jobs, NamedJob{Name: name, Job: job})
	}
	return newPipeline(PipelineOptions{
		Stages:    b.opts.Stages,
		Variables: b.opts.Variables,
		Default:   b.opts.Default,
		Workflow:  b.opts.Workflow,
		Jobs:      jobs,
	})
}

func newPipeline(opts PipelineOptions) (*Pipeline, error) {
	jobs := make(map[string]Job, len(opts.Jobs))
	for i := range opts.Jobs {
		job := opts.Jobs[i]
		if job.Name == "" {
			return nil, errors.New("gitlab job name is required")
		}
		if _, exists := jobs[job.Name]; exists {
			return nil, fmt.Errorf("duplicate gitlab job %q", job.Name)
		}
		jobs[job.Name] = job.Job.clone()
	}
	return &Pipeline{
		stages:    append([]string(nil), opts.Stages...),
		variables: cloneStringMap(opts.Variables),
		defaults:  cloneDefaultConfig(opts.Default),
		jobs:      jobs,
		workflow:  cloneWorkflow(opts.Workflow),
	}, nil
}

func EmptyPipeline() *Pipeline {
	return &Pipeline{jobs: make(map[string]Job)}
}

func (p *Pipeline) Stages() []string {
	if p == nil {
		return nil
	}
	return append([]string(nil), p.stages...)
}

func (p *Pipeline) Variables() map[string]string {
	if p == nil {
		return nil
	}
	return cloneStringMap(p.variables)
}

func (p *Pipeline) Default() *DefaultConfig {
	if p == nil {
		return nil
	}
	return cloneDefaultConfig(p.defaults)
}

func (p *Pipeline) Workflow() *Workflow {
	if p == nil {
		return nil
	}
	return cloneWorkflow(p.workflow)
}

func (p *Pipeline) Job(name string) (Job, bool) {
	if p == nil {
		return Job{}, false
	}
	job, ok := p.jobs[name]
	if !ok {
		return Job{}, false
	}
	return job.clone(), true
}

func (p *Pipeline) Jobs() map[string]Job {
	if p == nil {
		return nil
	}
	out := make(map[string]Job, len(p.jobs))
	for name := range p.jobs {
		job := p.jobs[name]
		out[name] = job.clone()
	}
	return out
}

func (p *Pipeline) JobNames() []string {
	if p == nil {
		return nil
	}
	names := make([]string, 0, len(p.jobs))
	for name := range p.jobs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (p *Pipeline) JobCount() int {
	if p == nil {
		return 0
	}
	return len(p.jobs)
}

func (p *Pipeline) HasNeed(jobName, dependency string) bool {
	job, ok := p.Job(jobName)
	return ok && job.HasNeed(dependency)
}

// DefaultConfig represents default job configuration.
type DefaultConfig struct {
	Image *ImageConfig `yaml:"image,omitempty"`
}

// ImageConfig represents GitLab CI image configuration.
type ImageConfig struct {
	Name       string   `yaml:"name,omitempty"`
	Entrypoint []string `yaml:"entrypoint,omitempty"`
}

// MarshalYAML emits the short string form when entrypoint is empty.
func (img ImageConfig) MarshalYAML() (any, error) {
	if len(img.Entrypoint) == 0 {
		return img.Name, nil
	}

	type imageAlias ImageConfig
	return imageAlias(img), nil
}

// IDToken represents GitLab CI OIDC token configuration.
type IDToken struct {
	Aud string `yaml:"aud"`
}

// Secret represents GitLab CI secret configuration.
type Secret struct {
	Vault     *VaultSecret `yaml:"vault,omitempty"`
	VaultPath string       `yaml:"-"`
	File      bool         `yaml:"file,omitempty"`
}

// MarshalYAML emits the short vault syntax when configured.
func (s Secret) MarshalYAML() (any, error) {
	if s.VaultPath != "" {
		type secretShorthand struct {
			Vault string `yaml:"vault"`
			File  bool   `yaml:"file,omitempty"`
		}

		return secretShorthand{
			Vault: s.VaultPath,
			File:  s.File,
		}, nil
	}

	type secretAlias Secret
	return secretAlias(s), nil
}

// VaultSecret represents a secret from HashiCorp Vault.
type VaultSecret struct {
	Engine *VaultEngine `yaml:"engine,omitempty"`
	Path   string       `yaml:"path,omitempty"`
	Field  string       `yaml:"field,omitempty"`
}

// VaultEngine represents Vault secrets engine configuration.
type VaultEngine struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

// Cache represents GitLab CI cache configuration.
type Cache struct {
	Key    string   `yaml:"key"`
	Paths  []string `yaml:"paths"`
	Policy string   `yaml:"policy,omitempty"`
}

// JobNeed represents a job dependency.
type JobNeed struct {
	Job       string `yaml:"job"`
	Optional  bool   `yaml:"optional,omitempty"`
	Artifacts *bool  `yaml:"artifacts,omitempty"`
}

// Rule represents a job or workflow rule.
type Rule struct {
	If      string   `yaml:"if,omitempty"`
	When    string   `yaml:"when,omitempty"`
	Changes []string `yaml:"changes,omitempty"`
}

// Artifacts represents job artifacts.
type Artifacts struct {
	Paths     []string `yaml:"paths,omitempty"`
	ExpireIn  string   `yaml:"expire_in,omitempty"`
	Reports   *Reports `yaml:"reports,omitempty"`
	Name      string   `yaml:"name,omitempty"`
	Untracked bool     `yaml:"untracked,omitempty"`
	When      string   `yaml:"when,omitempty"`
	ExposeAs  string   `yaml:"expose_as,omitempty"`
}

// Reports represents artifact reports.
type Reports struct {
	Terraform []string `yaml:"terraform,omitempty"`
	JUnit     []string `yaml:"junit,omitempty"`
	Cobertura []string `yaml:"cobertura,omitempty"`
}

// Workflow controls when pipelines are created.
type Workflow struct {
	rules []Rule
}

func NewWorkflow(rules []Rule) *Workflow {
	if len(rules) == 0 {
		return nil
	}
	return &Workflow{rules: cloneRules(rules)}
}

func (w *Workflow) Rules() []Rule {
	if w == nil {
		return nil
	}
	return cloneRules(w.rules)
}

func (w *Workflow) MarshalYAML() (any, error) {
	return struct {
		Rules []Rule `yaml:"rules,omitempty"`
	}{
		Rules: cloneRules(w.rules),
	}, nil
}

// ToYAML converts the pipeline to YAML.
func (p *Pipeline) ToYAML() ([]byte, error) {
	result := make(map[string]any)
	if p == nil {
		result["stages"] = []string(nil)
		return yaml.Marshal(result)
	}
	result["stages"] = p.stages

	if len(p.variables) > 0 {
		result["variables"] = p.variables
	}

	if p.defaults != nil {
		result["default"] = p.defaults
	}

	if p.workflow != nil {
		result["workflow"] = p.workflow
	}

	for _, name := range p.JobNames() {
		result[name] = p.jobs[name]
	}

	return yaml.Marshal(result)
}

type JobOptions struct {
	Stage         string
	Image         *ImageConfig
	Script        []string
	BeforeScript  []string
	AfterScript   []string
	Variables     map[string]string
	Needs         []JobNeed
	Rules         []Rule
	Artifacts     *Artifacts
	Cache         *Cache
	Secrets       map[string]*Secret
	IDTokens      map[string]*IDToken
	When          string
	AllowFailure  bool
	Tags          []string
	ResourceGroup string
}

// Job represents a GitLab CI job.
type Job struct {
	stage         string
	image         *ImageConfig
	script        []string
	beforeScript  []string
	afterScript   []string
	variables     map[string]string
	needs         []JobNeed
	rules         []Rule
	artifacts     *Artifacts
	cache         *Cache
	secrets       map[string]*Secret
	idTokens      map[string]*IDToken
	when          string
	allowFailure  bool
	tags          []string
	resourceGroup string
}

func NewJob(opts JobOptions) (Job, error) {
	if opts.Stage == "" {
		return Job{}, errors.New("gitlab job stage is required")
	}
	if len(opts.Script) == 0 {
		return Job{}, errors.New("gitlab job script is required")
	}
	return Job{
		stage:         opts.Stage,
		image:         cloneImageConfig(opts.Image),
		script:        append([]string(nil), opts.Script...),
		beforeScript:  append([]string(nil), opts.BeforeScript...),
		afterScript:   append([]string(nil), opts.AfterScript...),
		variables:     cloneStringMap(opts.Variables),
		needs:         cloneNeeds(opts.Needs),
		rules:         cloneRules(opts.Rules),
		artifacts:     cloneArtifacts(opts.Artifacts),
		cache:         cloneCache(opts.Cache),
		secrets:       cloneSecrets(opts.Secrets),
		idTokens:      cloneIDTokens(opts.IDTokens),
		when:          opts.When,
		allowFailure:  opts.AllowFailure,
		tags:          append([]string(nil), opts.Tags...),
		resourceGroup: opts.ResourceGroup,
	}, nil
}

func (j Job) Stage() string { return j.stage }

func (j Job) Image() *ImageConfig { return cloneImageConfig(j.image) }

func (j Job) Script() []string { return append([]string(nil), j.script...) }

func (j Job) BeforeScript() []string { return append([]string(nil), j.beforeScript...) }

func (j Job) AfterScript() []string { return append([]string(nil), j.afterScript...) }

func (j Job) Variables() map[string]string { return cloneStringMap(j.variables) }

func (j Job) Needs() []JobNeed { return cloneNeeds(j.needs) }

func (j Job) Rules() []Rule { return cloneRules(j.rules) }

func (j Job) Artifacts() *Artifacts { return cloneArtifacts(j.artifacts) }

func (j Job) Cache() *Cache { return cloneCache(j.cache) }

func (j Job) Secrets() map[string]*Secret { return cloneSecrets(j.secrets) }

func (j Job) IDTokens() map[string]*IDToken { return cloneIDTokens(j.idTokens) }

func (j Job) When() string { return j.when }

func (j Job) AllowFailure() bool { return j.allowFailure }

func (j Job) Tags() []string { return append([]string(nil), j.tags...) }

func (j Job) ResourceGroup() string { return j.resourceGroup }

func (j Job) HasNeed(name string) bool {
	for _, need := range j.needs {
		if need.Job == name {
			return true
		}
	}
	return false
}

func (j Job) clone() Job {
	return Job{
		stage:         j.stage,
		image:         cloneImageConfig(j.image),
		script:        append([]string(nil), j.script...),
		beforeScript:  append([]string(nil), j.beforeScript...),
		afterScript:   append([]string(nil), j.afterScript...),
		variables:     cloneStringMap(j.variables),
		needs:         cloneNeeds(j.needs),
		rules:         cloneRules(j.rules),
		artifacts:     cloneArtifacts(j.artifacts),
		cache:         cloneCache(j.cache),
		secrets:       cloneSecrets(j.secrets),
		idTokens:      cloneIDTokens(j.idTokens),
		when:          j.when,
		allowFailure:  j.allowFailure,
		tags:          append([]string(nil), j.tags...),
		resourceGroup: j.resourceGroup,
	}
}

func (j Job) MarshalYAML() (any, error) {
	return struct {
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
	}{
		Stage:         j.stage,
		Image:         cloneImageConfig(j.image),
		Script:        append([]string(nil), j.script...),
		BeforeScript:  append([]string(nil), j.beforeScript...),
		AfterScript:   append([]string(nil), j.afterScript...),
		Variables:     cloneStringMap(j.variables),
		Needs:         cloneNeeds(j.needs),
		Rules:         cloneRules(j.rules),
		Artifacts:     cloneArtifacts(j.artifacts),
		Cache:         cloneCache(j.cache),
		Secrets:       cloneSecrets(j.secrets),
		IDTokens:      cloneIDTokens(j.idTokens),
		When:          j.when,
		AllowFailure:  j.allowFailure,
		Tags:          append([]string(nil), j.tags...),
		ResourceGroup: j.resourceGroup,
	}, nil
}

func cloneDefaultConfig(in *DefaultConfig) *DefaultConfig {
	if in == nil {
		return nil
	}
	return &DefaultConfig{Image: cloneImageConfig(in.Image)}
}

func cloneImageConfig(in *ImageConfig) *ImageConfig {
	if in == nil {
		return nil
	}
	return &ImageConfig{Name: in.Name, Entrypoint: append([]string(nil), in.Entrypoint...)}
}

func cloneWorkflow(in *Workflow) *Workflow {
	if in == nil {
		return nil
	}
	return &Workflow{rules: cloneRules(in.rules)}
}

func cloneCache(in *Cache) *Cache {
	if in == nil {
		return nil
	}
	return &Cache{Key: in.Key, Paths: append([]string(nil), in.Paths...), Policy: in.Policy}
}

func cloneNeeds(in []JobNeed) []JobNeed {
	return append([]JobNeed(nil), in...)
}

func cloneRules(in []Rule) []Rule {
	if len(in) == 0 {
		return nil
	}
	out := make([]Rule, len(in))
	for i, rule := range in {
		out[i] = rule
		out[i].Changes = append([]string(nil), rule.Changes...)
	}
	return out
}

func cloneArtifacts(in *Artifacts) *Artifacts {
	if in == nil {
		return nil
	}
	out := *in
	out.Paths = append([]string(nil), in.Paths...)
	if in.Reports != nil {
		out.Reports = &Reports{
			Terraform: append([]string(nil), in.Reports.Terraform...),
			JUnit:     append([]string(nil), in.Reports.JUnit...),
			Cobertura: append([]string(nil), in.Reports.Cobertura...),
		}
	}
	return &out
}

func cloneSecrets(in map[string]*Secret) map[string]*Secret {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]*Secret, len(in))
	for name, secret := range in {
		if secret == nil {
			continue
		}
		clone := *secret
		if secret.Vault != nil {
			clone.Vault = &VaultSecret{
				Path:  secret.Vault.Path,
				Field: secret.Vault.Field,
			}
			if secret.Vault.Engine != nil {
				clone.Vault.Engine = &VaultEngine{Name: secret.Vault.Engine.Name, Path: secret.Vault.Engine.Path}
			}
		}
		out[name] = &clone
	}
	return out
}

func cloneIDTokens(in map[string]*IDToken) map[string]*IDToken {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]*IDToken, len(in))
	for name, token := range in {
		if token == nil {
			continue
		}
		clone := *token
		out[name] = &clone
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	maps.Copy(out, in)
	return out
}
