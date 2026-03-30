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
	return exprfast.New(ctx).Attr(attrs, name)
}

func evalObjectStringAttrs(objExpr *hclsyntax.ObjectConsExpr, ctx *hcl.EvalContext) map[string]string {
	return exprfast.New(ctx).ObjectStringAttrs(objExpr)
}

func evalLiteralString(expr hcl.Expression) (string, bool) {
	return exprfast.New(nil).String(expr)
}

func evalAttrValue(attr *hcl.Attribute) (cty.Value, hcl.Diagnostics) {
	if attr == nil {
		return cty.NilVal, nil
	}

	return attr.Expr.Value(nil)
}
