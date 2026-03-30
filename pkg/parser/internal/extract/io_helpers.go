package extract

import (
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2"
)

func evalContentStringAttr(content *hcl.BodyContent, name string) (string, bool) {
	return contentStringAttr(content, name)
}

func contentStringAttr(content *hcl.BodyContent, name string) (string, bool) {
	attr, ok := content.Attributes[name]
	if !ok {
		return "", false
	}

	return evalLiteralString(attr.Expr)
}

func tfvarsReadDiagnostic(path string, err error) hcl.Diagnostics {
	return hcl.Diagnostics{&hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "Failed to read tfvars file",
		Detail:   fmt.Sprintf("Could not read %s: %v", path, err),
	}}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
