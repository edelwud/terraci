package config

import "github.com/edelwud/terraci/pkg/ci"

type Image = ci.Image

// Config contains GitHub Actions specific settings.
type Config struct {
	RunsOn      string            `yaml:"runs_on" json:"runs_on" jsonschema:"description=GitHub Actions runner label (e.g. ubuntu-latest),default=ubuntu-latest"`
	Container   *Image            `yaml:"container,omitempty" json:"container,omitempty" jsonschema:"description=Container image to run jobs in (optional)"`
	Env         map[string]string `yaml:"env,omitempty" json:"env,omitempty" jsonschema:"description=Workflow-level environment variables"`
	PlanOnly    bool              `yaml:"plan_only" json:"plan_only" jsonschema:"description=Generate only plan jobs (no apply jobs),default=false"`
	Permissions map[string]string `yaml:"permissions,omitempty" json:"permissions,omitempty" jsonschema:"description=Workflow-level permissions (e.g. id-token: write for OIDC)"`
	JobDefaults *JobDefaults      `yaml:"job_defaults,omitempty" json:"job_defaults,omitempty" jsonschema:"description=Default settings applied to all jobs"`
	Overwrites  []JobOverwrite    `yaml:"overwrites,omitempty" json:"overwrites,omitempty" jsonschema:"description=Job-level overrides for plan or apply jobs"`
}

type JobDefaults struct {
	RunsOn      string            `yaml:"runs_on,omitempty" json:"runs_on,omitempty" jsonschema:"description=Override runner label"`
	Container   *Image            `yaml:"container,omitempty" json:"container,omitempty" jsonschema:"description=Container image for all jobs"`
	Env         map[string]string `yaml:"env,omitempty" json:"env,omitempty" jsonschema:"description=Additional environment variables"`
	If          string            `yaml:"if,omitempty" json:"if,omitempty" jsonschema:"description=GitHub Actions job condition"`
	Environment string            `yaml:"environment,omitempty" json:"environment,omitempty" jsonschema:"description=GitHub Actions environment name"`
	StepsBefore []ConfigStep      `yaml:"steps_before,omitempty" json:"steps_before,omitempty" jsonschema:"description=Extra steps before terraform commands"`
	StepsAfter  []ConfigStep      `yaml:"steps_after,omitempty" json:"steps_after,omitempty" jsonschema:"description=Extra steps after terraform commands"`
}

type JobOverwrite struct {
	Type        JobOverwriteType  `yaml:"type" json:"type" jsonschema:"description=Type of jobs to override (plan\\, apply\\, or contributed job name),required"`
	RunsOn      string            `yaml:"runs_on,omitempty" json:"runs_on,omitempty" jsonschema:"description=Override runner label"`
	Container   *Image            `yaml:"container,omitempty" json:"container,omitempty" jsonschema:"description=Container image override"`
	Env         map[string]string `yaml:"env,omitempty" json:"env,omitempty" jsonschema:"description=Additional environment variables"`
	If          string            `yaml:"if,omitempty" json:"if,omitempty" jsonschema:"description=GitHub Actions job condition"`
	Environment string            `yaml:"environment,omitempty" json:"environment,omitempty" jsonschema:"description=GitHub Actions environment name"`
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

type JobOverwriteType string

const (
	OverwriteTypePlan  JobOverwriteType = "plan"
	OverwriteTypeApply JobOverwriteType = "apply"
)
