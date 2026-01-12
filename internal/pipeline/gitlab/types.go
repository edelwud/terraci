// Package gitlab provides GitLab CI pipeline generation
package gitlab

import (
	"sort"

	"go.yaml.in/yaml/v4"
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
