package pipeline

// ScriptConfig controls terraform script generation for plan/apply jobs.
type ScriptConfig struct {
	TerraformBinary string
	InitEnabled     bool
	PlanEnabled     bool
	AutoApprove     bool
	DetailedPlan    bool // true when MR/PR integration needs plan.txt + plan.json
}

// PlanScript generates the terraform plan commands and artifact paths for a module.
func (sc ScriptConfig) PlanScript(modulePath string) (script, artifactPaths []string) {
	script = append(script, "cd "+modulePath)
	if sc.InitEnabled {
		script = append(script, "${TERRAFORM_BINARY} init")
	}

	artifactPaths = []string{modulePath + "/plan.tfplan"}

	if sc.DetailedPlan {
		script = append(script,
			"(${TERRAFORM_BINARY} plan -out=plan.tfplan -detailed-exitcode 2>&1 || echo $? > .tf_exit) | tee plan.txt",
			"${TERRAFORM_BINARY} show -json plan.tfplan > plan.json",
			`TF_EXIT=$(cat .tf_exit 2>/dev/null || echo 0); rm -f .tf_exit; if [ "$TF_EXIT" -eq 2 ]; then exit 0; else exit "$TF_EXIT"; fi`,
		)
		artifactPaths = append(artifactPaths,
			modulePath+"/plan.txt",
			modulePath+"/plan.json")
	} else {
		script = append(script, "${TERRAFORM_BINARY} plan -out=plan.tfplan")
	}

	return script, artifactPaths
}

// ApplyScript generates the terraform apply commands for a module.
func (sc ScriptConfig) ApplyScript(modulePath string) []string {
	script := []string{"cd " + modulePath}
	if sc.InitEnabled {
		script = append(script, "${TERRAFORM_BINARY} init")
	}

	switch {
	case sc.PlanEnabled:
		script = append(script, "${TERRAFORM_BINARY} apply plan.tfplan")
	case sc.AutoApprove:
		script = append(script, "${TERRAFORM_BINARY} apply -auto-approve")
	default:
		script = append(script, "${TERRAFORM_BINARY} apply")
	}

	return script
}
