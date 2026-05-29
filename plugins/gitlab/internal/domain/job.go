package domain

import "errors"

// DefaultConfig represents default job configuration.
type DefaultConfig struct {
	Image *ImageConfig `yaml:"image,omitempty"`
}

// ImageConfig represents GitLab CI image configuration.
type ImageConfig struct {
	Name       string   `yaml:"name,omitempty"`
	Entrypoint []string `yaml:"entrypoint,omitempty"`
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
