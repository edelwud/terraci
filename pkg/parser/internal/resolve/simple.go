package resolve

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/parser/internal/exprfast"
)

func (r Resolver) resolveSimple(pathExpr hcl.Expression, evalCtx *hcl.EvalContext) ([]string, error) {
	if path, ok := exprfast.New(evalCtx).String(pathExpr); ok {
		log.WithField("path", path).Debug("resolved simple path")
		return []string{path}, nil
	}

	log.WithField("reason", "evaluation failed").Debug("falling back to template extraction")
	return extractPathTemplate(pathExpr, evalCtx)
}
