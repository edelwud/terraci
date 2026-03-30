package resolve

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/pkg/log"
)

func (r Resolver) resolveForEach(forEachExpr, pathExpr hcl.Expression, evalCtx *hcl.EvalContext) ([]string, error) {
	forEachVal, diags := forEachExpr.Value(evalCtx)
	if diags.HasErrors() {
		log.WithField("reason", "for_each evaluation failed").Debug("falling back to template extraction")
		return extractPathTemplate(pathExpr, evalCtx)
	}

	var paths []string
	for it := forEachVal.ElementIterator(); it.Next(); {
		k, v := it.Element()

		eachKey, eachValue := k, v
		if forEachVal.Type().IsSetType() || forEachVal.Type().IsTupleType() || forEachVal.Type().IsListType() {
			eachKey = v
		}

		iterCtx := evalCtx.NewChild()
		iterCtx.Variables = map[string]cty.Value{
			"each": cty.ObjectVal(map[string]cty.Value{
				"key":   eachKey,
				"value": eachValue,
			}),
		}

		pathVal, diags := pathExpr.Value(iterCtx)
		if !diags.HasErrors() && pathVal.Type() == cty.String {
			log.WithField("path", pathVal.AsString()).Debug("resolved for_each path")
			paths = append(paths, pathVal.AsString())
		}
	}

	return paths, nil
}
