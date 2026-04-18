package config

import "github.com/edelwud/terraci/pkg/ci"

// Type aliases for shared types keep the public config surface stable.
type Image = ci.Image
type MRCommentConfig = ci.MRCommentConfig

// Config contains GitLab CI specific settings.
type Config struct {
	Image        Image             `yaml:"image" json:"image" jsonschema:"description=Docker image for terraform jobs,default=hashicorp/terraform:1.6"`
	StagesPrefix string            `yaml:"stages_prefix" json:"stages_prefix" jsonschema:"description=Prefix for stage names (produces: {prefix}-plan-0\\, {prefix}-apply-0\\, etc.),default=deploy"`
	Parallelism  int               `yaml:"parallelism" json:"parallelism" jsonschema:"description=Maximum parallel jobs per stage,minimum=1,default=5"`
	Variables    map[string]string `yaml:"variables,omitempty" json:"variables,omitempty" jsonschema:"description=Global pipeline variables"`
	PlanOnly     bool              `yaml:"plan_only" json:"plan_only" jsonschema:"description=Generate only plan jobs (no apply jobs),default=false"`
	AutoApprove  bool              `yaml:"auto_approve" json:"auto_approve" jsonschema:"description=Auto-approve applies (skip manual confirmation),default=false"`
	CacheEnabled bool              `yaml:"cache_enabled" json:"cache_enabled" jsonschema:"description=Enable caching of .terraform directory,default=true"`
	Cache        *CacheConfig      `yaml:"cache,omitempty" json:"cache,omitempty" jsonschema:"description=Advanced GitLab cache configuration for terraform jobs"`
	Rules        []Rule            `yaml:"rules,omitempty" json:"rules,omitempty" jsonschema:"description=Workflow rules for conditional pipeline execution"`
	JobDefaults  *JobDefaults      `yaml:"job_defaults,omitempty" json:"job_defaults,omitempty" jsonschema:"description=Default settings applied to all jobs"`
	Overwrites   []JobOverwrite    `yaml:"overwrites,omitempty" json:"overwrites,omitempty" jsonschema:"description=Job-level overrides for plan or apply jobs"`
	MR           *MRConfig         `yaml:"mr,omitempty" json:"mr,omitempty" jsonschema:"description=Merge request integration settings"`
}

// CacheConfig defines advanced GitLab CI cache configuration.
type CacheConfig struct {
	Enabled *bool    `yaml:"enabled,omitempty" json:"enabled,omitempty" jsonschema:"description=Enable GitLab cache for terraform jobs"`
	Key     string   `yaml:"key,omitempty" json:"key,omitempty" jsonschema:"description=Cache key template. Supports placeholders: {module_path}\\, {service}\\, {environment}\\, {region}\\, {module}"`
	Paths   []string `yaml:"paths,omitempty" json:"paths,omitempty" jsonschema:"description=Cache path templates. Supports placeholders: {module_path}\\, {service}\\, {environment}\\, {region}\\, {module}"`
	Policy  string   `yaml:"policy,omitempty" json:"policy,omitempty" jsonschema:"description=GitLab cache policy,enum=pull,enum=push,enum=pull-push"`
}

// GetImage returns the configured image.
func (g *Config) GetImage() Image {
	if g == nil {
		return Image{}
	}

	return g.Image
}

// MRConfig contains settings for MR integration.
type MRConfig struct {
	Comment    *MRCommentConfig  `yaml:"comment,omitempty" json:"comment,omitempty" jsonschema:"description=MR comment configuration"`
	Labels     []string          `yaml:"labels,omitempty" json:"labels,omitempty" jsonschema:"description=Labels to add to MR (supports placeholders: {service}\\, {environment}\\, {region}\\, {module})"`
	SummaryJob *SummaryJobConfig `yaml:"summary_job,omitempty" json:"summary_job,omitempty" jsonschema:"description=Summary job configuration"`
}

// SummaryJobConfig contains settings for the summary job.
type SummaryJobConfig struct {
	Image *Image   `yaml:"image,omitempty" json:"image,omitempty" jsonschema:"description=Docker image for summary job (must contain terraci)"`
	Tags  []string `yaml:"tags,omitempty" json:"tags,omitempty" jsonschema:"description=Runner tags for summary job"`
}

// Rule represents workflow or job rules from config input.
type Rule struct {
	If      string   `yaml:"if,omitempty" json:"if,omitempty"`
	When    string   `yaml:"when,omitempty" json:"when,omitempty"`
	Changes []string `yaml:"changes,omitempty" json:"changes,omitempty"`
}

// IDToken represents configured OIDC token settings.
type IDToken struct {
	Aud string `yaml:"aud" json:"aud"`
}

// VaultEngine represents configured Vault engine settings.
type VaultEngine struct {
	Name string `yaml:"name" json:"name"`
	Path string `yaml:"path" json:"path"`
}

// JobConfig is a shared read-only view over job defaults and overwrites.
type JobConfig interface {
	GetImage() *Image
	GetIDTokens() map[string]IDToken
	GetSecrets() map[string]CfgSecret
	GetBeforeScript() []string
	GetAfterScript() []string
	GetArtifacts() *ArtifactsConfig
	GetTags() []string
	GetRules() []Rule
	GetVariables() map[string]string
}

// JobDefaults defines default settings for all generated jobs.
type JobDefaults struct {
	Image        *Image               `yaml:"image,omitempty" json:"image,omitempty" jsonschema:"description=Docker image override for all jobs"`
	IDTokens     map[string]IDToken   `yaml:"id_tokens,omitempty" json:"id_tokens,omitempty" jsonschema:"description=OIDC tokens for cloud provider authentication"`
	Secrets      map[string]CfgSecret `yaml:"secrets,omitempty" json:"secrets,omitempty" jsonschema:"description=Secrets from external secret managers"`
	BeforeScript []string             `yaml:"before_script,omitempty" json:"before_script,omitempty" jsonschema:"description=Commands to run before each job"`
	AfterScript  []string             `yaml:"after_script,omitempty" json:"after_script,omitempty" jsonschema:"description=Commands to run after each job"`
	Artifacts    *ArtifactsConfig     `yaml:"artifacts,omitempty" json:"artifacts,omitempty" jsonschema:"description=GitLab CI artifacts configuration"`
	Tags         []string             `yaml:"tags,omitempty" json:"tags,omitempty" jsonschema:"description=GitLab runner tags"`
	Rules        []Rule               `yaml:"rules,omitempty" json:"rules,omitempty" jsonschema:"description=Job-level rules"`
	Variables    map[string]string    `yaml:"variables,omitempty" json:"variables,omitempty" jsonschema:"description=Additional variables"`
}

func (j *JobDefaults) GetImage() *Image                 { return j.Image }
func (j *JobDefaults) GetIDTokens() map[string]IDToken  { return j.IDTokens }
func (j *JobDefaults) GetSecrets() map[string]CfgSecret { return j.Secrets }
func (j *JobDefaults) GetBeforeScript() []string        { return j.BeforeScript }
func (j *JobDefaults) GetAfterScript() []string         { return j.AfterScript }
func (j *JobDefaults) GetArtifacts() *ArtifactsConfig   { return j.Artifacts }
func (j *JobDefaults) GetTags() []string                { return j.Tags }
func (j *JobDefaults) GetRules() []Rule                 { return j.Rules }
func (j *JobDefaults) GetVariables() map[string]string  { return j.Variables }

// JobOverwriteType defines the type of jobs to override.
type JobOverwriteType string

const (
	OverwriteTypePlan  JobOverwriteType = "plan"
	OverwriteTypeApply JobOverwriteType = "apply"
)

// JobOverwrite defines job-level overrides for plan or apply jobs.
type JobOverwrite struct {
	Type         JobOverwriteType     `yaml:"type" json:"type" jsonschema:"description=Type of jobs to override (plan\\, apply\\, or contributed job name),required"`
	Image        *Image               `yaml:"image,omitempty" json:"image,omitempty" jsonschema:"description=Docker image override for matching jobs"`
	IDTokens     map[string]IDToken   `yaml:"id_tokens,omitempty" json:"id_tokens,omitempty" jsonschema:"description=OIDC tokens for matching jobs"`
	Secrets      map[string]CfgSecret `yaml:"secrets,omitempty" json:"secrets,omitempty" jsonschema:"description=Secrets for matching jobs"`
	BeforeScript []string             `yaml:"before_script,omitempty" json:"before_script,omitempty" jsonschema:"description=Commands to run before matching jobs"`
	AfterScript  []string             `yaml:"after_script,omitempty" json:"after_script,omitempty" jsonschema:"description=Commands to run after matching jobs"`
	Artifacts    *ArtifactsConfig     `yaml:"artifacts,omitempty" json:"artifacts,omitempty" jsonschema:"description=Artifacts configuration for matching jobs"`
	Tags         []string             `yaml:"tags,omitempty" json:"tags,omitempty" jsonschema:"description=Runner tags for matching jobs"`
	Rules        []Rule               `yaml:"rules,omitempty" json:"rules,omitempty" jsonschema:"description=Job-level rules for matching jobs"`
	Variables    map[string]string    `yaml:"variables,omitempty" json:"variables,omitempty" jsonschema:"description=Variables for matching jobs"`
}

func (j *JobOverwrite) GetImage() *Image                 { return j.Image }
func (j *JobOverwrite) GetIDTokens() map[string]IDToken  { return j.IDTokens }
func (j *JobOverwrite) GetSecrets() map[string]CfgSecret { return j.Secrets }
func (j *JobOverwrite) GetBeforeScript() []string        { return j.BeforeScript }
func (j *JobOverwrite) GetAfterScript() []string         { return j.AfterScript }
func (j *JobOverwrite) GetArtifacts() *ArtifactsConfig   { return j.Artifacts }
func (j *JobOverwrite) GetTags() []string                { return j.Tags }
func (j *JobOverwrite) GetRules() []Rule                 { return j.Rules }
func (j *JobOverwrite) GetVariables() map[string]string  { return j.Variables }

// ArtifactsConfig defines GitLab CI artifacts configuration.
type ArtifactsConfig struct {
	Paths     []string         `yaml:"paths,omitempty" json:"paths,omitempty" jsonschema:"description=File/directory paths to include as artifacts"`
	ExpireIn  string           `yaml:"expire_in,omitempty" json:"expire_in,omitempty" jsonschema:"description=How long to keep artifacts (e.g. '1 day'\\, '1 week')"`
	Reports   *ArtifactReports `yaml:"reports,omitempty" json:"reports,omitempty" jsonschema:"description=Artifact reports configuration"`
	Name      string           `yaml:"name,omitempty" json:"name,omitempty" jsonschema:"description=Artifact archive name"`
	Untracked bool             `yaml:"untracked,omitempty" json:"untracked,omitempty" jsonschema:"description=Include all untracked files"`
	When      string           `yaml:"when,omitempty" json:"when,omitempty" jsonschema:"description=When to upload artifacts,enum=on_success,enum=on_failure,enum=always"`
	ExposeAs  string           `yaml:"expose_as,omitempty" json:"expose_as,omitempty" jsonschema:"description=Makes artifacts available in MR UI"`
}

// ArtifactReports defines artifact reports configuration.
type ArtifactReports struct {
	Terraform []string `yaml:"terraform,omitempty" json:"terraform,omitempty" jsonschema:"description=Terraform report paths"`
	JUnit     []string `yaml:"junit,omitempty" json:"junit,omitempty" jsonschema:"description=JUnit report paths"`
	Cobertura []string `yaml:"cobertura,omitempty" json:"cobertura,omitempty" jsonschema:"description=Cobertura coverage report paths"`
}

// CfgSecret defines a CI/CD secret in the config file.
type CfgSecret struct {
	Vault *CfgVaultSecret `yaml:"vault,omitempty" json:"vault,omitempty" jsonschema:"description=HashiCorp Vault secret configuration"`
	File  bool            `yaml:"file,omitempty" json:"file,omitempty" jsonschema:"description=Write secret to a file"`
}

// CfgVaultSecret defines a secret from HashiCorp Vault in the config file.
type CfgVaultSecret struct {
	Engine    *VaultEngine `yaml:"engine,omitempty" json:"engine,omitempty" jsonschema:"description=Vault secrets engine configuration"`
	Path      string       `yaml:"path,omitempty" json:"path,omitempty" jsonschema:"description=Path to the secret in Vault"`
	Field     string       `yaml:"field,omitempty" json:"field,omitempty" jsonschema:"description=Field to extract from the secret"`
	Shorthand string       `yaml:"-" jsonschema:"-"`
}

// UnmarshalYAML supports both shorthand and full object syntax.
func (v *CfgVaultSecret) UnmarshalYAML(unmarshal func(any) error) error {
	var shorthand string
	if err := unmarshal(&shorthand); err == nil {
		v.Shorthand = shorthand
		return nil
	}

	type vaultSecretAlias CfgVaultSecret
	var alias vaultSecretAlias
	if err := unmarshal(&alias); err != nil {
		return err
	}

	*v = CfgVaultSecret(alias)
	return nil
}
