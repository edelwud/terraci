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

// NewPlanOperation creates a typed terraform plan operation plus the artifact
// that must restore plan files at their original workspace-relative paths.
func (sc ScriptConfig) NewPlanOperation(jobName, modulePath string) (op Operation, artifact Artifact) {
	op = Operation{
		Type: OperationTypeTerraformPlan,
		Terraform: &TerraformOperation{
			ModulePath:   modulePath,
			InitEnabled:  sc.InitEnabled,
			PlanFile:     modulePath + "/plan.tfplan",
			DetailedPlan: sc.DetailedPlan,
		},
	}

	artifactPaths := []string{op.Terraform.PlanFile}
	if sc.DetailedPlan {
		op.Terraform.PlanTextFile = modulePath + "/plan.txt"
		op.Terraform.PlanJSONFile = modulePath + "/plan.json"
		artifactPaths = append(artifactPaths, op.Terraform.PlanTextFile, op.Terraform.PlanJSONFile)
	}

	return op, PlanArtifact(jobName, artifactPaths)
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
