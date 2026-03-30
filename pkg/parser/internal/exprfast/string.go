package exprfast

import (
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

func evalString(expr hcl.Expression, ctx *hcl.EvalContext) (string, bool) {
	if val, ok := literalString(expr); ok {
		return val, true
	}
	if val, ok := templateString(expr, ctx); ok {
		return val, true
	}

	val, diags := expr.Value(ctx)
	if diags.HasErrors() || val.Type() != cty.String {
		return "", false
	}
	return val.AsString(), true
}

func literalString(expr hcl.Expression) (string, bool) {
	switch e := expr.(type) {
	case *hclsyntax.LiteralValueExpr:
		if e.Val.Type() != cty.String {
			return "", false
		}
		return e.Val.AsString(), true
	case *hclsyntax.TemplateWrapExpr:
		return literalString(e.Wrapped)
	default:
		return "", false
	}
}

func templateString(expr hcl.Expression, ctx *hcl.EvalContext) (string, bool) {
	template, ok := expr.(*hclsyntax.TemplateExpr)
	if !ok {
		return "", false
	}

	var builder strings.Builder
	evaluator := New(ctx)
	for _, part := range template.Parts {
		val, ok := evaluator.String(part)
		if !ok {
			return "", false
		}
		builder.WriteString(val)
	}
	return builder.String(), true
}
