package config

import "github.com/edelwud/terraci/pkg/ciprovider"

type Image = ciprovider.Image
type MRCommentConfig = ciprovider.MRCommentConfig

// Config contains GitHub Actions specific settings.
type Config struct {
	TerraformBinary string            `yaml:"terraform_binary" json:"terraform_binary" jsonschema:"description=Terraform/OpenTofu binary to use,enum=terraform,enum=tofu,default=terraform"`
	RunsOn          string            `yaml:"runs_on" json:"runs_on" jsonschema:"description=GitHub Actions runner label (e.g. ubuntu-latest),default=ubuntu-latest"`
	Container       *Image            `yaml:"container,omitempty" json:"container,omitempty" jsonschema:"description=Container image to run jobs in (optional)"`
	Env             map[string]string `yaml:"env,omitempty" json:"env,omitempty" jsonschema:"description=Workflow-level environment variables"`
	PlanEnabled     bool              `yaml:"plan_enabled" json:"plan_enabled" jsonschema:"description=Enable terraform plan jobs,default=true"`
	PlanOnly        bool              `yaml:"plan_only" json:"plan_only" jsonschema:"description=Generate only plan jobs (no apply jobs),default=false"`
	AutoApprove     bool              `yaml:"auto_approve" json:"auto_approve" jsonschema:"description=Auto-approve applies (skip environment protection),default=false"`
	InitEnabled     bool              `yaml:"init_enabled" json:"init_enabled" jsonschema:"description=Automatically run terraform init,default=true"`
	Permissions     map[string]string `yaml:"permissions,omitempty" json:"permissions,omitempty" jsonschema:"description=Workflow-level permissions (e.g. id-token: write for OIDC)"`
	JobDefaults     *JobDefaults      `yaml:"job_defaults,omitempty" json:"job_defaults,omitempty" jsonschema:"description=Default settings applied to all jobs"`
	Overwrites      []JobOverwrite    `yaml:"overwrites,omitempty" json:"overwrites,omitempty" jsonschema:"description=Job-level overrides for plan or apply jobs"`
	PR              *PRConfig         `yaml:"pr,omitempty" json:"pr,omitempty" jsonschema:"description=Pull request integration settings"`
}

type JobDefaults struct {
	RunsOn      string            `yaml:"runs_on,omitempty" json:"runs_on,omitempty" jsonschema:"description=Override runner label"`
	Container   *Image            `yaml:"container,omitempty" json:"container,omitempty" jsonschema:"description=Container image for all jobs"`
	Env         map[string]string `yaml:"env,omitempty" json:"env,omitempty" jsonschema:"description=Additional environment variables"`
	StepsBefore []ConfigStep      `yaml:"steps_before,omitempty" json:"steps_before,omitempty" jsonschema:"description=Extra steps before terraform commands"`
	StepsAfter  []ConfigStep      `yaml:"steps_after,omitempty" json:"steps_after,omitempty" jsonschema:"description=Extra steps after terraform commands"`
}

type JobOverwrite struct {
	Type        JobOverwriteType  `yaml:"type" json:"type" jsonschema:"description=Type of jobs to override (plan\\, apply\\, or contributed job name),required"`
	RunsOn      string            `yaml:"runs_on,omitempty" json:"runs_on,omitempty" jsonschema:"description=Override runner label"`
	Container   *Image            `yaml:"container,omitempty" json:"container,omitempty" jsonschema:"description=Container image override"`
	Env         map[string]string `yaml:"env,omitempty" json:"env,omitempty" jsonschema:"description=Additional environment variables"`
	StepsBefore []ConfigStep      `yaml:"steps_before,omitempty" json:"steps_before,omitempty" jsonschema:"description=Extra steps before terraform commands"`
	StepsAfter  []ConfigStep      `yaml:"steps_after,omitempty" json:"steps_after,omitempty" jsonschema:"description=Extra steps after terraform commands"`
}

//nolint:revive // ConfigStep keeps the public config vocabulary explicit.
type ConfigStep struct {
	Name string            `yaml:"name,omitempty" json:"name,omitempty" jsonschema:"description=Step display name"`
	Uses string            `yaml:"uses,omitempty" json:"uses,omitempty" jsonschema:"description=GitHub Action reference"`
	With map[string]string `yaml:"with,omitempty" json:"with,omitempty" jsonschema:"description=Action inputs"`
	Run  string            `yaml:"run,omitempty" json:"run,omitempty" jsonschema:"description=Shell command to run"`
	Env  map[string]string `yaml:"env,omitempty" json:"env,omitempty" jsonschema:"description=Step environment variables"`
}

type PRConfig struct {
	Comment    *MRCommentConfig  `yaml:"comment,omitempty" json:"comment,omitempty" jsonschema:"description=PR comment configuration"`
	SummaryJob *SummaryJobConfig `yaml:"summary_job,omitempty" json:"summary_job,omitempty" jsonschema:"description=Summary job configuration"`
}

type SummaryJobConfig struct {
	RunsOn string `yaml:"runs_on,omitempty" json:"runs_on,omitempty" jsonschema:"description=Runner label for summary job"`
}

type JobOverwriteType string

const (
	OverwriteTypePlan  JobOverwriteType = "plan"
	OverwriteTypeApply JobOverwriteType = "apply"
)
