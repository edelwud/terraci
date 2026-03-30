package resolve

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/edelwud/terraci/pkg/parser/internal/exprfast"
)

func extractPathTemplate(expr hcl.Expression, ctx *hcl.EvalContext) ([]string, error) {
	return exprfast.ExtractTemplate(expr, ctx)
}
