package pipeline

// ScriptConfig captures the knobs that influence how Build() populates each
// TerraformOperation. The struct does not render shell — see
// pkg/pipeline/cishell for the default shell renderer.
type ScriptConfig struct {
	InitEnabled bool
	PlanEnabled bool
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
		Type: OperationTypeTerraformPlan,
		Terraform: &TerraformOperation{
			ModulePath:   modulePath,
			InitEnabled:  sc.InitEnabled,
			PlanFile:     PlanBinaryPath(modulePath),
			DetailedPlan: outputs.Detailed(),
		},
	}

	resources := []ResourceSpec{
		PlanResource(ResourceKindPlanBinary, modulePath, op.Terraform.PlanFile),
	}
	if outputs.Text {
		op.Terraform.PlanTextFile = PlanTextPath(modulePath)
		resources = append(resources, PlanResource(ResourceKindPlanText, modulePath, op.Terraform.PlanTextFile))
	}
	if outputs.JSON {
		op.Terraform.PlanJSONFile = PlanJSONPath(modulePath)
		resources = append(resources, PlanResource(ResourceKindPlanJSON, modulePath, op.Terraform.PlanJSONFile))
	}

	return op, resources, PlanArtifact(jobName, resourcePaths(resources))
}

// NewApplyOperation creates a typed terraform apply operation.
func (sc ScriptConfig) NewApplyOperation(modulePath string) Operation {
	return Operation{
		Type: OperationTypeTerraformApply,
		Terraform: &TerraformOperation{
			ModulePath:  modulePath,
			InitEnabled: sc.InitEnabled,
			PlanFile:    PlanBinaryPath(modulePath),
			UsePlanFile: sc.PlanEnabled,
		},
	}
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
