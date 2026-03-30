package parser

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
)

func variableDefaultSchema() *hcl.BodySchema {
	return &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{{Name: "default"}},
	}
}

func tfvarsReadDiagnostic(path string, err error) hcl.Diagnostics {
	return hcl.Diagnostics{&hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "Failed to read tfvars file",
		Detail:   fmt.Sprintf("Could not read %s: %v", path, err),
	}}
}
