package parser

import (
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// addDiags appends diagnostics to the parsed module.
func (pm *ParsedModule) addDiags(diags hcl.Diagnostics) {
	pm.Diagnostics = append(pm.Diagnostics, diags...)
}

// evalContentStringAttr evaluates a named attribute from HCL content as a string.
func evalContentStringAttr(content *hcl.BodyContent, name string) (string, bool) {
	attr, ok := content.Attributes[name]
	if !ok {
		return "", false
	}
	return evalStringExpr(attr.Expr, nil)
}

// evalStringExpr evaluates an expression as a string value.
func evalStringExpr(expr hcl.Expression, ctx *hcl.EvalContext) (string, bool) {
	val, diags := expr.Value(ctx)
	if diags.HasErrors() || val.Type() != cty.String {
		return "", false
	}
	return val.AsString(), true
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
