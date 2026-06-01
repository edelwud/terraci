package pipeline

import (
	"maps"

	"github.com/edelwud/terraci/pkg/terraformrun"
)

// TerraformJobConfig captures Terraform/OpenTofu runtime intent copied into
// Terraform jobs. The struct does not render shell — see pkg/pipeline/cishell.
type TerraformJobConfig struct {
	binary      terraformrun.Binary
	initEnabled bool
	env         map[string]string
}

// TerraformJobConfigOptions configures NewTerraformJobConfig.
type TerraformJobConfigOptions struct {
	Binary      string
	InitEnabled bool
	Env         map[string]string
}

// NewTerraformJobConfig creates immutable Terraform job runtime config.
func NewTerraformJobConfig(opts TerraformJobConfigOptions) (TerraformJobConfig, error) {
	binary, err := terraformrun.ParseBinary(opts.Binary)
	if err != nil {
		return TerraformJobConfig{}, err
	}
	return TerraformJobConfig{
		binary:      binary,
		initEnabled: opts.InitEnabled,
		env:         maps.Clone(opts.Env),
	}, nil
}

type PlanOutputs struct {
	Text bool
	JSON bool
}

func (o PlanOutputs) Detailed() bool {
	return o.Text || o.JSON
}

// NewPlanOperation creates a typed terraform plan operation plus the resources
// and artifact that must restore plan files at their original
// workspace-relative paths.
func (c TerraformJobConfig) NewPlanOperation(jobName, modulePath string, outputs PlanOutputs) (Operation, []ResourceSpec, Artifact) {
	op := Operation{
		typ: OperationTypeTerraformPlan,
		terraform: &TerraformOperation{
			binary:       c.binary,
			kind:         OperationTypeTerraformPlan,
			modulePath:   modulePath,
			initEnabled:  c.initEnabled,
			planFile:     PlanBinaryPath(modulePath),
			detailedPlan: outputs.Detailed(),
		},
	}

	resources := []ResourceSpec{
		PlanResource(ResourceKindPlanBinary, modulePath, op.terraform.planFile),
	}
	if outputs.Text {
		op.terraform.planTextFile = PlanTextPath(modulePath)
		resources = append(resources, PlanResource(ResourceKindPlanText, modulePath, op.terraform.planTextFile))
	}
	if outputs.JSON {
		op.terraform.planJSONFile = PlanJSONPath(modulePath)
		resources = append(resources, PlanResource(ResourceKindPlanJSON, modulePath, op.terraform.planJSONFile))
	}

	return op, resources, PlanArtifact(jobName, resourcePaths(resources))
}

// NewApplyOperation creates a typed terraform apply operation.
func (c TerraformJobConfig) NewApplyOperation(modulePath string, usePlanFile bool) Operation {
	return Operation{
		typ: OperationTypeTerraformApply,
		terraform: &TerraformOperation{
			binary:      c.binary,
			kind:        OperationTypeTerraformApply,
			modulePath:  modulePath,
			initEnabled: c.initEnabled,
			planFile:    PlanBinaryPath(modulePath),
			usePlanFile: usePlanFile,
		},
	}
}

// TerraformEnv returns a defensive copy of Terraform job environment.
func (c TerraformJobConfig) TerraformEnv() map[string]string {
	if len(c.env) == 0 {
		return nil
	}
	return maps.Clone(c.env)
}

func resourcePaths(resources []ResourceSpec) []string {
	if len(resources) == 0 {
		return nil
	}

	paths := make([]string, 0, len(resources))
	for _, resource := range resources {
		paths = append(paths, resource.Path)
	}
	return paths
}
