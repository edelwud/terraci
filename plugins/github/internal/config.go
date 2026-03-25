package githubci

// Config contains GitHub Actions specific settings
type Config struct {
	// TerraformBinary is the terraform binary to use (e.g., "terraform", "tofu")
	TerraformBinary string `yaml:"terraform_binary" json:"terraform_binary" jsonschema:"description=Terraform/OpenTofu binary to use,enum=terraform,enum=tofu,default=terraform"`
	// RunsOn specifies the runner label(s) for jobs
	RunsOn string `yaml:"runs_on" json:"runs_on" jsonschema:"description=GitHub Actions runner label (e.g. ubuntu-latest),default=ubuntu-latest"`
	// Container optionally runs jobs in a container
	Container *Image `yaml:"container,omitempty" json:"container,omitempty" jsonschema:"description=Container image to run jobs in (optional)"`
	// Env sets workflow-level environment variables
	Env map[string]string `yaml:"env,omitempty" json:"env,omitempty" jsonschema:"description=Workflow-level environment variables"`
	// PlanEnabled enables terraform plan jobs
	PlanEnabled bool `yaml:"plan_enabled" json:"plan_enabled" jsonschema:"description=Enable terraform plan jobs,default=true"`
	// PlanOnly generates only plan jobs without apply jobs
	PlanOnly bool `yaml:"plan_only" json:"plan_only" jsonschema:"description=Generate only plan jobs (no apply jobs),default=false"`
	// AutoApprove skips manual approval for apply
	AutoApprove bool `yaml:"auto_approve" json:"auto_approve" jsonschema:"description=Auto-approve applies (skip environment protection),default=false"`
	// InitEnabled automatically runs terraform init
	InitEnabled bool `yaml:"init_enabled" json:"init_enabled" jsonschema:"description=Automatically run terraform init,default=true"`
	// Permissions sets workflow-level permissions (e.g., id-token: write)
	Permissions map[string]string `yaml:"permissions,omitempty" json:"permissions,omitempty" jsonschema:"description=Workflow-level permissions (e.g. id-token: write for OIDC)"`
	// JobDefaults defines default settings for all jobs
	JobDefaults *JobDefaults `yaml:"job_defaults,omitempty" json:"job_defaults,omitempty" jsonschema:"description=Default settings applied to all jobs"`
	// Overwrites defines job-level overrides for plan and apply jobs
	Overwrites []JobOverwrite `yaml:"overwrites,omitempty" json:"overwrites,omitempty" jsonschema:"description=Job-level overrides for plan or apply jobs"`
	// PR contains pull request integration settings
	PR *PRConfig `yaml:"pr,omitempty" json:"pr,omitempty" jsonschema:"description=Pull request integration settings"`
}

// JobDefaults defines default settings for all GitHub Actions jobs
type JobDefaults struct {
	// RunsOn overrides the runner label for all jobs
	RunsOn string `yaml:"runs_on,omitempty" json:"runs_on,omitempty" jsonschema:"description=Override runner label"`
	// Container runs jobs in a container
	Container *Image `yaml:"container,omitempty" json:"container,omitempty" jsonschema:"description=Container image for all jobs"`
	// Env sets additional environment variables for all jobs
	Env map[string]string `yaml:"env,omitempty" json:"env,omitempty" jsonschema:"description=Additional environment variables"`
	// StepsBefore are extra steps to run before terraform commands
	StepsBefore []ConfigStep `yaml:"steps_before,omitempty" json:"steps_before,omitempty" jsonschema:"description=Extra steps before terraform commands"`
	// StepsAfter are extra steps to run after terraform commands
	StepsAfter []ConfigStep `yaml:"steps_after,omitempty" json:"steps_after,omitempty" jsonschema:"description=Extra steps after terraform commands"`
}

// JobOverwrite defines job-level overrides for plan or apply jobs
type JobOverwrite struct {
	// Type specifies which jobs to override: "plan" or "apply"
	Type JobOverwriteType `yaml:"type" json:"type" jsonschema:"description=Type of jobs to override,enum=plan,enum=apply,required"`
	// RunsOn overrides the runner label
	RunsOn string `yaml:"runs_on,omitempty" json:"runs_on,omitempty" jsonschema:"description=Override runner label"`
	// Container runs jobs in a container
	Container *Image `yaml:"container,omitempty" json:"container,omitempty" jsonschema:"description=Container image override"`
	// Env sets additional environment variables
	Env map[string]string `yaml:"env,omitempty" json:"env,omitempty" jsonschema:"description=Additional environment variables"`
	// StepsBefore are extra steps to run before terraform commands
	StepsBefore []ConfigStep `yaml:"steps_before,omitempty" json:"steps_before,omitempty" jsonschema:"description=Extra steps before terraform commands"`
	// StepsAfter are extra steps to run after terraform commands
	StepsAfter []ConfigStep `yaml:"steps_after,omitempty" json:"steps_after,omitempty" jsonschema:"description=Extra steps after terraform commands"`
}

// ConfigStep represents a step in a GitHub Actions job (for job_defaults)
type ConfigStep struct {
	// Name is the step display name
	Name string `yaml:"name,omitempty" json:"name,omitempty" jsonschema:"description=Step display name"`
	// Uses references a GitHub Action (e.g., actions/checkout@v4)
	Uses string `yaml:"uses,omitempty" json:"uses,omitempty" jsonschema:"description=GitHub Action reference"`
	// With provides inputs to the action
	With map[string]string `yaml:"with,omitempty" json:"with,omitempty" jsonschema:"description=Action inputs"`
	// Run is a shell command
	Run string `yaml:"run,omitempty" json:"run,omitempty" jsonschema:"description=Shell command to run"`
	// Env sets environment variables for this step
	Env map[string]string `yaml:"env,omitempty" json:"env,omitempty" jsonschema:"description=Step environment variables"`
}

// PRConfig contains settings for PR/MR integration (used by GitHub provider)
type PRConfig struct {
	// Comment enables PR comment with plan summary
	Comment *MRCommentConfig `yaml:"comment,omitempty" json:"comment,omitempty" jsonschema:"description=PR comment configuration"`
	// SummaryJob configures the summary job that posts PR comments
	SummaryJob *SummaryJobConfig `yaml:"summary_job,omitempty" json:"summary_job,omitempty" jsonschema:"description=Summary job configuration"`
}

// SummaryJobConfig contains settings for the GitHub Actions summary job
type SummaryJobConfig struct {
	// RunsOn specifies the runner label for the summary job
	RunsOn string `yaml:"runs_on,omitempty" json:"runs_on,omitempty" jsonschema:"description=Runner label for summary job"`
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

// JobOverwriteType defines the type of jobs to override
type JobOverwriteType string

const (
	// OverwriteTypePlan applies to plan jobs only
	OverwriteTypePlan JobOverwriteType = "plan"
	// OverwriteTypeApply applies to apply jobs only
	OverwriteTypeApply JobOverwriteType = "apply"
)

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
