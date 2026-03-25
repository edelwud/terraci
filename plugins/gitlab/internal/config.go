package gitlabci

// Config contains GitLab CI specific settings
type Config struct {
	// TerraformBinary is the terraform binary to use (e.g., "terraform", "tofu")
	TerraformBinary string `yaml:"terraform_binary" json:"terraform_binary" jsonschema:"description=Terraform/OpenTofu binary to use,enum=terraform,enum=tofu,default=terraform"`
	// Image is the Docker image for terraform jobs (in default section)
	// Supports both string format ("hashicorp/terraform:1.6") and object format with entrypoint
	Image Image `yaml:"image" json:"image" jsonschema:"description=Docker image for terraform jobs,default=hashicorp/terraform:1.6"`
	// StagesPrefix is the prefix for stage names (e.g., "deploy" -> "deploy-0", "deploy-1")
	StagesPrefix string `yaml:"stages_prefix" json:"stages_prefix" jsonschema:"description=Prefix for stage names (produces: {prefix}-plan-0\\, {prefix}-apply-0\\, etc.),default=deploy"`
	// Parallelism limits concurrent jobs per stage
	Parallelism int `yaml:"parallelism" json:"parallelism" jsonschema:"description=Maximum parallel jobs per stage,minimum=1,default=5"`
	// Variables to set in the pipeline (global variables section)
	Variables map[string]string `yaml:"variables,omitempty" json:"variables,omitempty" jsonschema:"description=Global pipeline variables"`
	// PlanEnabled enables terraform plan stage
	PlanEnabled bool `yaml:"plan_enabled" json:"plan_enabled" jsonschema:"description=Enable terraform plan stage,default=true"`
	// PlanOnly generates only plan jobs without apply jobs
	PlanOnly bool `yaml:"plan_only" json:"plan_only" jsonschema:"description=Generate only plan jobs (no apply jobs),default=false"`
	// AutoApprove skips manual approval for apply
	AutoApprove bool `yaml:"auto_approve" json:"auto_approve" jsonschema:"description=Auto-approve applies (skip manual confirmation),default=false"`
	// CacheEnabled enables caching of .terraform directory
	CacheEnabled bool `yaml:"cache_enabled" json:"cache_enabled" jsonschema:"description=Enable caching of .terraform directory,default=true"`
	// InitEnabled automatically runs terraform init after cd to module directory
	InitEnabled bool `yaml:"init_enabled" json:"init_enabled" jsonschema:"description=Automatically run terraform init after cd to module directory,default=true"`
	// Rules defines workflow-level rules for conditional pipeline execution
	Rules []Rule `yaml:"rules,omitempty" json:"rules,omitempty" jsonschema:"description=Workflow rules for conditional pipeline execution"`
	// JobDefaults defines default settings for all jobs (applied before overwrites)
	JobDefaults *JobDefaults `yaml:"job_defaults,omitempty" json:"job_defaults,omitempty" jsonschema:"description=Default settings applied to all jobs"`
	// Overwrites defines job-level overrides for plan and apply jobs
	Overwrites []JobOverwrite `yaml:"overwrites,omitempty" json:"overwrites,omitempty" jsonschema:"description=Job-level overrides for plan or apply jobs"`
	// MR contains merge request integration settings
	MR *MRConfig `yaml:"mr,omitempty" json:"mr,omitempty" jsonschema:"description=Merge request integration settings"`
}

// GetImage returns the configured image
func (g *Config) GetImage() Image {
	if g == nil {
		return Image{}
	}
	return g.Image
}

// MRConfig contains settings for MR integration
type MRConfig struct {
	// Comment enables MR comment with plan summary (auto-detected in MR pipelines)
	Comment *MRCommentConfig `yaml:"comment,omitempty" json:"comment,omitempty" jsonschema:"description=MR comment configuration"`
	// Labels to add to MR, supports placeholders: {service}, {environment}, {region}, {module}
	Labels []string `yaml:"labels,omitempty" json:"labels,omitempty" jsonschema:"description=Labels to add to MR (supports placeholders: {service}\\, {environment}\\, {region}\\, {module})"`
	// SummaryJob configures the summary job that posts MR comments
	SummaryJob *SummaryJobConfig `yaml:"summary_job,omitempty" json:"summary_job,omitempty" jsonschema:"description=Summary job configuration"`
}

// SummaryJobConfig contains settings for the summary job
type SummaryJobConfig struct {
	// Image for the summary job (must contain terraci binary)
	Image *Image `yaml:"image,omitempty" json:"image,omitempty" jsonschema:"description=Docker image for summary job (must contain terraci)"`
	// Tags for the summary job runner
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty" jsonschema:"description=Runner tags for summary job"`
}

// MRCommentConfig contains settings for MR/PR comments.
// Shared by both GitLab and GitHub plugins.
type MRCommentConfig struct {
	// Enabled enables MR comments (default: true when in MR pipeline)
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty" jsonschema:"description=Enable MR comments,default=true"`
	// OnChangesOnly only comment when there are changes (default: false)
	OnChangesOnly bool `yaml:"on_changes_only,omitempty" json:"on_changes_only,omitempty" jsonschema:"description=Only comment when there are changes"`
	// IncludeDetails includes full plan output in collapsible sections
	IncludeDetails bool `yaml:"include_details,omitempty" json:"include_details,omitempty" jsonschema:"description=Include full plan output in expandable sections,default=true"`
}

// JobConfig is an interface for job configuration (defaults and overwrites)
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

// JobDefaults defines default settings for all generated jobs
type JobDefaults struct {
	// Image overrides the Docker image for all jobs
	Image *Image `yaml:"image,omitempty" json:"image,omitempty" jsonschema:"description=Docker image override for all jobs"`
	// IDTokens sets OIDC tokens for all jobs
	IDTokens map[string]IDToken `yaml:"id_tokens,omitempty" json:"id_tokens,omitempty" jsonschema:"description=OIDC tokens for cloud provider authentication"`
	// Secrets sets secrets for all jobs
	Secrets map[string]CfgSecret `yaml:"secrets,omitempty" json:"secrets,omitempty" jsonschema:"description=Secrets from external secret managers"`
	// BeforeScript sets before_script for all jobs
	BeforeScript []string `yaml:"before_script,omitempty" json:"before_script,omitempty" jsonschema:"description=Commands to run before each job"`
	// AfterScript sets after_script for all jobs
	AfterScript []string `yaml:"after_script,omitempty" json:"after_script,omitempty" jsonschema:"description=Commands to run after each job"`
	// Artifacts sets artifacts configuration for all jobs
	Artifacts *ArtifactsConfig `yaml:"artifacts,omitempty" json:"artifacts,omitempty" jsonschema:"description=GitLab CI artifacts configuration"`
	// Tags sets runner tags for all jobs
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty" jsonschema:"description=GitLab runner tags"`
	// Rules sets job-level rules for all jobs
	Rules []Rule `yaml:"rules,omitempty" json:"rules,omitempty" jsonschema:"description=Job-level rules"`
	// Variables sets additional variables for all jobs
	Variables map[string]string `yaml:"variables,omitempty" json:"variables,omitempty" jsonschema:"description=Additional variables"`
}

// JobDefaults implements JobConfig
func (j *JobDefaults) GetImage() *Image                 { return j.Image }
func (j *JobDefaults) GetIDTokens() map[string]IDToken  { return j.IDTokens }
func (j *JobDefaults) GetSecrets() map[string]CfgSecret { return j.Secrets }
func (j *JobDefaults) GetBeforeScript() []string        { return j.BeforeScript }
func (j *JobDefaults) GetAfterScript() []string         { return j.AfterScript }
func (j *JobDefaults) GetArtifacts() *ArtifactsConfig   { return j.Artifacts }
func (j *JobDefaults) GetTags() []string                { return j.Tags }
func (j *JobDefaults) GetRules() []Rule                 { return j.Rules }
func (j *JobDefaults) GetVariables() map[string]string  { return j.Variables }

// JobOverwriteType defines the type of jobs to override
type JobOverwriteType string

const (
	// OverwriteTypePlan applies to plan jobs only
	OverwriteTypePlan JobOverwriteType = "plan"
	// OverwriteTypeApply applies to apply jobs only
	OverwriteTypeApply JobOverwriteType = "apply"
)

// JobOverwrite defines job-level overrides for plan or apply jobs
type JobOverwrite struct {
	// Type specifies which jobs to override: "plan" or "apply"
	Type JobOverwriteType `yaml:"type" json:"type" jsonschema:"description=Type of jobs to override,enum=plan,enum=apply,required"`
	// Image overrides the Docker image for matching jobs
	Image *Image `yaml:"image,omitempty" json:"image,omitempty" jsonschema:"description=Docker image override for matching jobs"`
	// IDTokens overrides OIDC tokens for matching jobs
	IDTokens map[string]IDToken `yaml:"id_tokens,omitempty" json:"id_tokens,omitempty" jsonschema:"description=OIDC tokens for matching jobs"`
	// Secrets overrides secrets for matching jobs
	Secrets map[string]CfgSecret `yaml:"secrets,omitempty" json:"secrets,omitempty" jsonschema:"description=Secrets for matching jobs"`
	// BeforeScript overrides before_script for matching jobs
	BeforeScript []string `yaml:"before_script,omitempty" json:"before_script,omitempty" jsonschema:"description=Commands to run before matching jobs"`
	// AfterScript overrides after_script for matching jobs
	AfterScript []string `yaml:"after_script,omitempty" json:"after_script,omitempty" jsonschema:"description=Commands to run after matching jobs"`
	// Artifacts overrides artifacts configuration for matching jobs
	Artifacts *ArtifactsConfig `yaml:"artifacts,omitempty" json:"artifacts,omitempty" jsonschema:"description=Artifacts configuration for matching jobs"`
	// Tags overrides runner tags for matching jobs
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty" jsonschema:"description=Runner tags for matching jobs"`
	// Rules sets job-level rules for matching jobs
	Rules []Rule `yaml:"rules,omitempty" json:"rules,omitempty" jsonschema:"description=Job-level rules for matching jobs"`
	// Variables overrides variables for matching jobs
	Variables map[string]string `yaml:"variables,omitempty" json:"variables,omitempty" jsonschema:"description=Variables for matching jobs"`
}

// JobOverwrite implements JobConfig
func (j *JobOverwrite) GetImage() *Image                 { return j.Image }
func (j *JobOverwrite) GetIDTokens() map[string]IDToken  { return j.IDTokens }
func (j *JobOverwrite) GetSecrets() map[string]CfgSecret { return j.Secrets }
func (j *JobOverwrite) GetBeforeScript() []string        { return j.BeforeScript }
func (j *JobOverwrite) GetAfterScript() []string         { return j.AfterScript }
func (j *JobOverwrite) GetArtifacts() *ArtifactsConfig   { return j.Artifacts }
func (j *JobOverwrite) GetTags() []string                { return j.Tags }
func (j *JobOverwrite) GetRules() []Rule                 { return j.Rules }
func (j *JobOverwrite) GetVariables() map[string]string  { return j.Variables }

// ArtifactsConfig defines GitLab CI artifacts configuration
type ArtifactsConfig struct {
	// Paths is a list of file/directory paths to include as artifacts
	Paths []string `yaml:"paths,omitempty" json:"paths,omitempty" jsonschema:"description=File/directory paths to include as artifacts"`
	// ExpireIn specifies how long artifacts should be kept (e.g., "1 day", "1 week")
	ExpireIn string `yaml:"expire_in,omitempty" json:"expire_in,omitempty" jsonschema:"description=How long to keep artifacts (e.g. '1 day'\\, '1 week')"`
	// Reports defines artifact reports (e.g., terraform)
	Reports *ArtifactReports `yaml:"reports,omitempty" json:"reports,omitempty" jsonschema:"description=Artifact reports configuration"`
	// Name is the artifact archive name
	Name string `yaml:"name,omitempty" json:"name,omitempty" jsonschema:"description=Artifact archive name"`
	// Untracked includes all untracked files
	Untracked bool `yaml:"untracked,omitempty" json:"untracked,omitempty" jsonschema:"description=Include all untracked files"`
	// When specifies when to upload artifacts: on_success, on_failure, always
	When string `yaml:"when,omitempty" json:"when,omitempty" jsonschema:"description=When to upload artifacts,enum=on_success,enum=on_failure,enum=always"`
	// ExposeAs makes artifacts available in MR UI
	ExposeAs string `yaml:"expose_as,omitempty" json:"expose_as,omitempty" jsonschema:"description=Makes artifacts available in MR UI"`
}

// ArtifactReports defines artifact reports configuration
type ArtifactReports struct {
	// Terraform report paths
	Terraform []string `yaml:"terraform,omitempty" json:"terraform,omitempty" jsonschema:"description=Terraform report paths"`
	// JUnit report paths
	JUnit []string `yaml:"junit,omitempty" json:"junit,omitempty" jsonschema:"description=JUnit report paths"`
	// Cobertura coverage report paths
	Cobertura []string `yaml:"cobertura,omitempty" json:"cobertura,omitempty" jsonschema:"description=Cobertura coverage report paths"`
}

// CfgSecret defines a CI/CD secret in the config file (from an external secret manager).
// This is the config-input type; the pipeline-output type is Secret in types.go.
type CfgSecret struct {
	// Vault configures HashiCorp Vault secret (can be string shorthand or object)
	Vault *CfgVaultSecret `yaml:"vault,omitempty" json:"vault,omitempty" jsonschema:"description=HashiCorp Vault secret configuration"`
	// File indicates if secret should be written to a file
	File bool `yaml:"file,omitempty" json:"file,omitempty" jsonschema:"description=Write secret to a file"`
}

// CfgVaultSecret defines a secret from HashiCorp Vault in the config file.
// Supports both full object syntax and string shorthand (path/to/secret/field@namespace).
type CfgVaultSecret struct {
	// Engine is the secrets engine (e.g., "kv-v2") - for full syntax
	Engine *VaultEngine `yaml:"engine,omitempty" json:"engine,omitempty" jsonschema:"description=Vault secrets engine configuration"`
	// Path is the path to the secret in Vault - for full syntax
	Path string `yaml:"path,omitempty" json:"path,omitempty" jsonschema:"description=Path to the secret in Vault"`
	// Field is the field to extract from the secret - for full syntax
	Field string `yaml:"field,omitempty" json:"field,omitempty" jsonschema:"description=Field to extract from the secret"`
	// Shorthand is the string shorthand format (path/to/secret/field@namespace)
	Shorthand string `yaml:"-" jsonschema:"-"`
}

// UnmarshalYAML implements custom unmarshaling for CfgVaultSecret to support string shorthand
func (v *CfgVaultSecret) UnmarshalYAML(unmarshal func(any) error) error {
	// Try string shorthand first
	var shorthand string
	if err := unmarshal(&shorthand); err == nil {
		v.Shorthand = shorthand
		return nil
	}

	// Try full object syntax
	type vaultSecretAlias CfgVaultSecret
	var alias vaultSecretAlias
	if err := unmarshal(&alias); err != nil {
		return err
	}
	*v = CfgVaultSecret(alias)
	return nil
}

// Image defines a Docker image configuration
// Supports both string format and object format with entrypoint
type Image struct {
	// Name is the image name (e.g., "hashicorp/terraform:1.6")
	Name string `yaml:"name,omitempty" json:"name,omitempty" jsonschema:"description=Docker image name"`
	// Entrypoint overrides the default entrypoint
	Entrypoint []string `yaml:"entrypoint,omitempty" json:"entrypoint,omitempty" jsonschema:"description=Override default entrypoint"`
}

// UnmarshalYAML implements custom unmarshaling for Image to support string shorthand
func (img *Image) UnmarshalYAML(unmarshal func(any) error) error {
	// Try string shorthand first (just image name)
	var shorthand string
	if err := unmarshal(&shorthand); err == nil {
		img.Name = shorthand
		return nil
	}

	// Try full object syntax
	type imageAlias Image
	var alias imageAlias
	if err := unmarshal(&alias); err != nil {
		return err
	}
	*img = Image(alias)
	return nil
}

// String returns the image name
func (img *Image) String() string {
	return img.Name
}

// HasEntrypoint returns true if entrypoint is configured
func (img *Image) HasEntrypoint() bool {
	return len(img.Entrypoint) > 0
}
