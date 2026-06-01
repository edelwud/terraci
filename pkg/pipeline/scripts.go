package pipeline

import "maps"

// ScriptConfig captures the knobs that influence how BuildProjectIR populates each
// TerraformOperation. The struct does not render shell — see
// pkg/pipeline/cishell for the default shell renderer.
type ScriptConfig struct {
	InitEnabled bool
	Env         map[string]string
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
func (sc ScriptConfig) NewPlanOperation(jobName, modulePath string, outputs PlanOutputs) (Operation, []ResourceSpec, Artifact) {
	op := Operation{
		typ: OperationTypeTerraformPlan,
		terraform: &TerraformOperation{
			kind:         OperationTypeTerraformPlan,
			modulePath:   modulePath,
			initEnabled:  sc.InitEnabled,
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
func (sc ScriptConfig) NewApplyOperation(modulePath string, usePlanFile bool) Operation {
	return Operation{
		typ: OperationTypeTerraformApply,
		terraform: &TerraformOperation{
			kind:        OperationTypeTerraformApply,
			modulePath:  modulePath,
			initEnabled: sc.InitEnabled,
			planFile:    PlanBinaryPath(modulePath),
			usePlanFile: usePlanFile,
		},
	}
}

// TerraformEnv returns a defensive copy of execution-level Terraform job environment.
func (sc ScriptConfig) TerraformEnv() map[string]string {
	if len(sc.Env) == 0 {
		return nil
	}
	return maps.Clone(sc.Env)
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
