package resolve

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/pkg/log"
)

func (r Resolver) resolveSimple(pathExpr hcl.Expression, evalCtx *hcl.EvalContext) ([]string, error) {
	pathVal, diags := pathExpr.Value(evalCtx)
	if !diags.HasErrors() && pathVal.Type() == cty.String {
		log.WithField("path", pathVal.AsString()).Debug("resolved simple path")
		return []string{pathVal.AsString()}, nil
	}

	log.WithField("reason", "evaluation failed").Debug("falling back to template extraction")
	return extractPathTemplate(pathExpr, evalCtx)
}
