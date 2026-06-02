package pipeline

import "github.com/edelwud/terraci/pkg/terraformrun"

// TerraformRuntime summarizes Terraform/OpenTofu runtime intent encoded in IR.
type TerraformRuntime struct {
	binary terraformrun.Binary
	mixed  bool
}

// TerraformRuntime returns Terraform/OpenTofu runtime intent found in IR jobs.
func (ir *IR) TerraformRuntime() (TerraformRuntime, bool) {
	if ir == nil {
		return TerraformRuntime{}, false
	}

	var runtime TerraformRuntime
	found := false
	for i := range ir.jobs {
		op := ir.jobs[i].operation.terraform
		if op == nil {
			continue
		}
		binary := normalizedBinary(op.binary)
		if !found {
			runtime.binary = binary
			found = true
			continue
		}
		if runtime.binary != binary {
			runtime.mixed = true
		}
	}
	return runtime, found
}

// Binary returns the Terraform-compatible executable name.
func (r TerraformRuntime) Binary() string { return r.binary.String() }

// Mixed reports whether Terraform jobs use more than one executable.
func (r TerraformRuntime) Mixed() bool { return r.mixed }

func normalizedBinary(binary terraformrun.Binary) terraformrun.Binary {
	if binary == "" {
		return terraformrun.BinaryTerraform
	}
	return binary
}
