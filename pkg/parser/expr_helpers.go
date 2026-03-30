package parser

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

func evalStringExpr(expr hcl.Expression, ctx *hcl.EvalContext) (string, bool) {
	val, diags := expr.Value(ctx)
	if diags.HasErrors() || val.Type() != cty.String {
		return "", false
	}
	return val.AsString(), true
}
