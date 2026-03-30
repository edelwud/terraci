package parser

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/edelwud/terraci/pkg/parser/internal/exprfast"
)

func evalStringExpr(expr hcl.Expression, ctx *hcl.EvalContext) (string, bool) {
	return exprfast.EvalString(expr, ctx)
}
