package resolve

import (
	"errors"

	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/pkg/log"
)

func (r Resolver) Resolve(
	ref Ref,
	modulePath string,
	locals, variables map[string]cty.Value,
) ([]string, error) {
	log.WithField("module", modulePath).WithField("remote_state", ref.Name).Debug("resolving workspace path")

	evalCtx := r.evalBuilder.Build(modulePath, locals, variables)

	pathExpr := findPathExpression(ref)
	if pathExpr == nil {
		return nil, errors.New("no key or prefix found in remote state config")
	}

	if ref.ForEach != nil {
		return r.resolveForEach(ref.ForEach, pathExpr, evalCtx)
	}

	return r.resolveSimple(pathExpr, evalCtx)
}
