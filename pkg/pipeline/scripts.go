package pipeline

// ScriptConfig captures the knobs that influence how Build() populates each
// TerraformOperation. The struct does not render shell — see
// pkg/pipeline/cishell for the default shell renderer.
type ScriptConfig struct {
	InitEnabled  bool
	PlanEnabled  bool
	AutoApprove  bool
	DetailedPlan bool // true when MR/PR integration needs plan.txt + plan.json
}

// NewPlanOperation creates a typed terraform plan operation plus the resources
// and artifact that must restore plan files at their original
// workspace-relative paths.
func (sc ScriptConfig) NewPlanOperation(jobName, modulePath string) (Operation, []ResourceSpec, Artifact) {
	op := Operation{
		Type: OperationTypeTerraformPlan,
		Terraform: &TerraformOperation{
			ModulePath:   modulePath,
			InitEnabled:  sc.InitEnabled,
			PlanFile:     modulePath + "/plan.tfplan",
			DetailedPlan: sc.DetailedPlan,
		},
	}

	resources := []ResourceSpec{
		PlanResource(ResourceKindPlanBinary, modulePath, op.Terraform.PlanFile),
	}
	if sc.DetailedPlan {
		op.Terraform.PlanTextFile = modulePath + "/plan.txt"
		op.Terraform.PlanJSONFile = modulePath + "/plan.json"
		resources = append(resources,
			PlanResource(ResourceKindPlanText, modulePath, op.Terraform.PlanTextFile),
			PlanResource(ResourceKindPlanJSON, modulePath, op.Terraform.PlanJSONFile),
		)
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
			PlanFile:    modulePath + "/plan.tfplan",
			UsePlanFile: sc.PlanEnabled,
			AutoApprove: sc.AutoApprove,
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
