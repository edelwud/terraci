package exprfast

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

type Evaluator struct {
	ctx *hcl.EvalContext
}

func New(ctx *hcl.EvalContext) Evaluator {
	return Evaluator{ctx: ctx}
}

func (e Evaluator) String(expr hcl.Expression) (string, bool) {
	return evalString(expr, e.ctx)
}

func (e Evaluator) Attr(attrs map[string]*hcl.Attribute, name string) (string, bool) {
	attr, ok := attrs[name]
	if !ok {
		return "", false
	}

	return e.String(attr.Expr)
}

func (e Evaluator) ContentAttr(content *hcl.BodyContent, name string) (string, bool) {
	if content == nil {
		return "", false
	}

	return e.Attr(content.Attributes, name)
}

func (e Evaluator) ObjectStringAttrs(objExpr *hclsyntax.ObjectConsExpr) map[string]string {
	values := make(map[string]string)
	if objExpr == nil {
		return values
	}

	for _, item := range objExpr.Items {
		key, ok := New(nil).String(item.KeyExpr)
		if !ok {
			continue
		}

		value, ok := e.String(item.ValueExpr)
		if !ok {
			continue
		}

		values[key] = value
	}

	return values
}

func EvalString(expr hcl.Expression, ctx *hcl.EvalContext) (string, bool) {
	return New(ctx).String(expr)
}

func ContentStringAttr(content *hcl.BodyContent, name string, ctx *hcl.EvalContext) (string, bool) {
	return New(ctx).ContentAttr(content, name)
}
