package config

import (
	"maps"

	"github.com/edelwud/terraci/pkg/ci"
)

type Image = ci.Image

// Config contains GitHub Actions specific settings.
type Config struct {
	RunsOn      string            `yaml:"runs_on" json:"runs_on" jsonschema:"description=GitHub Actions runner label (e.g. ubuntu-latest),default=ubuntu-latest"`
	Container   *Image            `yaml:"container,omitempty" json:"container,omitempty" jsonschema:"description=Container image to run jobs in (optional)"`
	Env         map[string]string `yaml:"env,omitempty" json:"env,omitempty" jsonschema:"description=Workflow-level environment variables"`
	Permissions map[string]string `yaml:"permissions,omitempty" json:"permissions,omitempty" jsonschema:"description=Workflow-level permissions (e.g. id-token: write for OIDC)"`
	JobDefaults *JobDefaults      `yaml:"job_defaults,omitempty" json:"job_defaults,omitempty" jsonschema:"description=Default settings applied to all jobs"`
	Overwrites  []JobOverwrite    `yaml:"overwrites,omitempty" json:"overwrites,omitempty" jsonschema:"description=Job-level overrides for plan or apply jobs"`
}

// Clone returns a deep copy of the GitHub Actions configuration.
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}
	out := *c
	out.Container = cloneImagePointer(c.Container)
	out.Env = maps.Clone(c.Env)
	out.Permissions = maps.Clone(c.Permissions)
	out.JobDefaults = cloneJobDefaults(c.JobDefaults)
	out.Overwrites = cloneJobOverwrites(c.Overwrites)
	return &out
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

func cloneJobDefaults(in *JobDefaults) *JobDefaults {
	if in == nil {
		return nil
	}
	out := *in
	out.Container = cloneImagePointer(in.Container)
	out.Env = maps.Clone(in.Env)
	out.StepsBefore = cloneConfigSteps(in.StepsBefore)
	out.StepsAfter = cloneConfigSteps(in.StepsAfter)
	return &out
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

func cloneJobOverwrites(in []JobOverwrite) []JobOverwrite {
	if len(in) == 0 {
		return nil
	}
	out := make([]JobOverwrite, len(in))
	for i := range in {
		out[i] = in[i]
		out[i].Container = cloneImagePointer(in[i].Container)
		out[i].Env = maps.Clone(in[i].Env)
		out[i].StepsBefore = cloneConfigSteps(in[i].StepsBefore)
		out[i].StepsAfter = cloneConfigSteps(in[i].StepsAfter)
	}
	return out
}

//nolint:revive // ConfigStep keeps the public config vocabulary explicit.
type ConfigStep struct {
	Name string            `yaml:"name,omitempty" json:"name,omitempty" jsonschema:"description=Step display name"`
	Uses string            `yaml:"uses,omitempty" json:"uses,omitempty" jsonschema:"description=GitHub Action reference"`
	With map[string]string `yaml:"with,omitempty" json:"with,omitempty" jsonschema:"description=Action inputs"`
	Run  string            `yaml:"run,omitempty" json:"run,omitempty" jsonschema:"description=Shell command to run"`
	Env  map[string]string `yaml:"env,omitempty" json:"env,omitempty" jsonschema:"description=Step environment variables"`
}

func cloneConfigSteps(in []ConfigStep) []ConfigStep {
	if len(in) == 0 {
		return nil
	}
	out := make([]ConfigStep, len(in))
	for i := range in {
		out[i] = in[i]
		out[i].With = maps.Clone(in[i].With)
		out[i].Env = maps.Clone(in[i].Env)
	}
	return out
}

func cloneImagePointer(in *Image) *Image {
	if in == nil {
		return nil
	}
	out := *in
	out.Entrypoint = append([]string(nil), in.Entrypoint...)
	return &out
}

type JobOverwriteType string

const (
	OverwriteTypePlan  JobOverwriteType = "plan"
	OverwriteTypeApply JobOverwriteType = "apply"
)
