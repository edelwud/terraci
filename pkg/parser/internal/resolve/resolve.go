package resolve

import (
	"errors"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/parser/internal/evalctx"
)

type Ref struct {
	Name    string
	Config  map[string]hcl.Expression
	ForEach hcl.Expression
}

type Resolver struct {
	evalBuilder evalctx.Builder
}

func NewResolver(evalBuilder evalctx.Builder) Resolver {
	return Resolver{evalBuilder: evalBuilder}
}

func (r Resolver) Resolve(
	ref Ref,
	modulePath string,
	locals, variables map[string]cty.Value,
) ([]string, error) {
	log.WithField("module", modulePath).WithField("remote_state", ref.Name).Debug("resolving workspace path")

	evalCtx := r.evalBuilder.Build(modulePath, locals, variables)

	pathExpr := r.findPathExpression(ref)
	if pathExpr == nil {
		return nil, errors.New("no key or prefix found in remote state config")
	}

	if ref.ForEach != nil {
		return r.resolveForEach(ref.ForEach, pathExpr, evalCtx)
	}

	return r.resolveSimple(pathExpr, evalCtx)
}

func (r Resolver) findPathExpression(ref Ref) hcl.Expression {
	if expr, ok := ref.Config["key"]; ok {
		return expr
	}
	if expr, ok := ref.Config["prefix"]; ok {
		return expr
	}
	return nil
}

func (r Resolver) resolveSimple(pathExpr hcl.Expression, evalCtx *hcl.EvalContext) ([]string, error) {
	pathVal, diags := pathExpr.Value(evalCtx)
	if !diags.HasErrors() && pathVal.Type() == cty.String {
		log.WithField("path", pathVal.AsString()).Debug("resolved simple path")
		return []string{pathVal.AsString()}, nil
	}

	log.WithField("reason", "evaluation failed").Debug("falling back to template extraction")
	return extractPathTemplate(pathExpr, evalCtx)
}

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

func extractPathTemplate(expr hcl.Expression, ctx *hcl.EvalContext) ([]string, error) {
	val, _ := expr.Value(ctx) //nolint:errcheck
	if val.IsKnown() && val.Type() == cty.String {
		return []string{val.AsString()}, nil
	}

	rng := expr.Range()
	if rng.Filename != "" {
		content, err := os.ReadFile(rng.Filename)
		if err == nil {
			start, end := rng.Start.Byte, rng.End.Byte
			if end <= len(content) {
				return []string{string(content[start:end])}, nil
			}
		}
	}

	return nil, errors.New("could not extract path template")
}
