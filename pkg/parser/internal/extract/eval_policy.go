package extract

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/pkg/parser/internal/exprfast"
)

func (c *Context) buildEvalContext() *hcl.EvalContext {
	return c.EvalBuilder.Build(c.Sink.Path(), c.Sink.Locals(), c.Sink.Variables())
}

func evalStringAttr(attrs map[string]*hcl.Attribute, name string, ctx *hcl.EvalContext) (string, bool) {
	attr, ok := attrs[name]
	if !ok {
		return "", false
	}

	return exprfast.EvalString(attr.Expr, ctx)
}

func evalObjectStringAttrs(objExpr *hclsyntax.ObjectConsExpr, ctx *hcl.EvalContext) map[string]string {
	values := make(map[string]string)
	for _, item := range objExpr.Items {
		key, ok := exprfast.EvalString(item.KeyExpr, nil)
		if !ok {
			continue
		}

		value, ok := exprfast.EvalString(item.ValueExpr, ctx)
		if !ok {
			continue
		}

		values[key] = value
	}

	return values
}

func evalLiteralString(expr hcl.Expression) (string, bool) {
	return exprfast.EvalString(expr, nil)
}

func evalAttrValue(attr *hcl.Attribute) (cty.Value, hcl.Diagnostics) {
	if attr == nil {
		return cty.NilVal, nil
	}

	return attr.Expr.Value(nil)
}
