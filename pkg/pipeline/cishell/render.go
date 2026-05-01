// Package cishell renders a pipeline.Operation into POSIX-shell command lines
// for CI providers that drive Terraform/OpenTofu through a shell. Providers
// that compose runs differently (e.g. an in-process tfexec runner) do not
// need this package — the IR carries everything required to drive Terraform
// directly via TerraformOperation.
package cishell

import (
	"fmt"
	"path/filepath"

	"github.com/edelwud/terraci/pkg/pipeline"
)

// RenderOperation converts a typed operation into shell command lines.
func RenderOperation(op pipeline.Operation) []string {
	switch op.Type {
	case pipeline.OperationTypeCommands:
		return append([]string(nil), op.Commands...)
	case pipeline.OperationTypeTerraformPlan:
		return renderTerraformPlan(op.Terraform)
	case pipeline.OperationTypeTerraformApply:
		return renderTerraformApply(op.Terraform)
	default:
		return nil
	}
}

func renderTerraformPlan(op *pipeline.TerraformOperation) []string {
	if op == nil {
		return nil
	}

	planFile := filepath.Base(op.PlanFile)
	script := []string{"cd " + op.ModulePath}
	if op.InitEnabled {
		script = append(script, "${TERRAFORM_BINARY} init")
	}

	if op.DetailedPlan {
		planText := filepath.Base(op.PlanTextFile)
		planJSON := filepath.Base(op.PlanJSONFile)
		script = append(script,
			fmt.Sprintf("(${TERRAFORM_BINARY} plan -out=%s -detailed-exitcode 2>&1 || echo $? > .tf_exit) | tee %s", planFile, planText),
			fmt.Sprintf("${TERRAFORM_BINARY} show -json %s > %s", planFile, planJSON),
			`TF_EXIT=$(cat .tf_exit 2>/dev/null || echo 0); rm -f .tf_exit; if [ "$TF_EXIT" -eq 2 ]; then exit 0; else exit "$TF_EXIT"; fi`,
		)
		return script
	}

	return append(script, "${TERRAFORM_BINARY} plan -out="+planFile)
}

func renderTerraformApply(op *pipeline.TerraformOperation) []string {
	if op == nil {
		return nil
	}

	script := []string{"cd " + op.ModulePath}
	if op.InitEnabled {
		script = append(script, "${TERRAFORM_BINARY} init")
	}

	switch {
	case op.UsePlanFile:
		script = append(script, "${TERRAFORM_BINARY} apply "+filepath.Base(op.PlanFile))
	case op.AutoApprove:
		script = append(script, "${TERRAFORM_BINARY} apply -auto-approve")
	default:
		script = append(script, "${TERRAFORM_BINARY} apply")
	}

	return script
}
