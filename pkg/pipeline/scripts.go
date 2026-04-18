package pipeline

import "fmt"

// ScriptConfig controls terraform operation generation.
type ScriptConfig struct {
	InitEnabled  bool
	PlanEnabled  bool
	AutoApprove  bool
	DetailedPlan bool // true when MR/PR integration needs plan.txt + plan.json
}

// NewPlanOperation creates a typed terraform plan operation plus artifact paths.
func (sc ScriptConfig) NewPlanOperation(modulePath string) (op Operation, artifactPaths []string) {
	op = Operation{
		Type: OperationTypeTerraformPlan,
		Terraform: &TerraformOperation{
			Kind:         OperationTypeTerraformPlan,
			ModulePath:   modulePath,
			InitEnabled:  sc.InitEnabled,
			PlanFile:     modulePath + "/plan.tfplan",
			DetailedPlan: sc.DetailedPlan,
		},
	}

	artifactPaths = []string{op.Terraform.PlanFile}
	if sc.DetailedPlan {
		op.Terraform.PlanTextFile = modulePath + "/plan.txt"
		op.Terraform.PlanJSONFile = modulePath + "/plan.json"
		artifactPaths = append(artifactPaths, op.Terraform.PlanTextFile, op.Terraform.PlanJSONFile)
	}

	return op, artifactPaths
}

// NewApplyOperation creates a typed terraform apply operation.
func (sc ScriptConfig) NewApplyOperation(modulePath string) Operation {
	return Operation{
		Type: OperationTypeTerraformApply,
		Terraform: &TerraformOperation{
			Kind:        OperationTypeTerraformApply,
			ModulePath:  modulePath,
			InitEnabled: sc.InitEnabled,
			PlanFile:    modulePath + "/plan.tfplan",
			UsePlanFile: sc.PlanEnabled,
			AutoApprove: sc.AutoApprove,
		},
	}
}

// RenderOperationScript converts a typed operation into shell commands for CI renderers.
func RenderOperationScript(op Operation) []string {
	switch op.Type {
	case OperationTypeCommands:
		return append([]string(nil), op.Commands...)
	case OperationTypeTerraformPlan:
		return renderTerraformPlan(op.Terraform)
	case OperationTypeTerraformApply:
		return renderTerraformApply(op.Terraform)
	default:
		return nil
	}
}

func renderTerraformPlan(op *TerraformOperation) []string {
	if op == nil {
		return nil
	}

	script := []string{"cd " + op.ModulePath}
	if op.InitEnabled {
		script = append(script, "${TERRAFORM_BINARY} init")
	}

	if op.DetailedPlan {
		script = append(script,
			fmt.Sprintf("(${TERRAFORM_BINARY} plan -out=%s -detailed-exitcode 2>&1 || echo $? > .tf_exit) | tee %s", "plan.tfplan", "plan.txt"),
			fmt.Sprintf("${TERRAFORM_BINARY} show -json %s > %s", "plan.tfplan", "plan.json"),
			`TF_EXIT=$(cat .tf_exit 2>/dev/null || echo 0); rm -f .tf_exit; if [ "$TF_EXIT" -eq 2 ]; then exit 0; else exit "$TF_EXIT"; fi`,
		)
		return script
	}

	return append(script, "${TERRAFORM_BINARY} plan -out=plan.tfplan")
}

func renderTerraformApply(op *TerraformOperation) []string {
	if op == nil {
		return nil
	}

	script := []string{"cd " + op.ModulePath}
	if op.InitEnabled {
		script = append(script, "${TERRAFORM_BINARY} init")
	}

	switch {
	case op.UsePlanFile:
		script = append(script, "${TERRAFORM_BINARY} apply plan.tfplan")
	case op.AutoApprove:
		script = append(script, "${TERRAFORM_BINARY} apply -auto-approve")
	default:
		script = append(script, "${TERRAFORM_BINARY} apply")
	}

	return script
}
