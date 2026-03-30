package parser

import (
	"errors"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/pkg/log"
)

// ResolveWorkspacePath resolves workspace paths from a remote state config
// using the module's locals, variables, and path-derived segment values.
func (p *Parser) ResolveWorkspacePath(ref *RemoteStateRef, modulePath string, locals, variables map[string]cty.Value) ([]string, error) {
	return newRemoteStateResolver(p.evalContextBuilder()).Resolve(ref, modulePath, locals, variables)
}

type remoteStateResolver struct {
	evalBuilder evalContextBuilder
}

func newRemoteStateResolver(evalBuilder evalContextBuilder) remoteStateResolver {
	return remoteStateResolver{evalBuilder: evalBuilder}
}

func (r remoteStateResolver) Resolve(
	ref *RemoteStateRef,
	modulePath string,
	locals, variables map[string]cty.Value,
) ([]string, error) {
	log.WithField("module", modulePath).WithField("remote_state", ref.Name).Debug("resolving workspace path")

	evalCtx := r.evalBuilder.build(modulePath, locals, variables)

	pathExpr := r.findPathExpression(ref)
	if pathExpr == nil {
		return nil, errors.New("no key or prefix found in remote state config")
	}

	if ref.ForEach != nil {
		return r.resolveForEach(ref.ForEach, pathExpr, evalCtx)
	}

	return r.resolveSimple(pathExpr, evalCtx)
}

// findPathExpression returns the key or prefix expression from remote state config.
func (r remoteStateResolver) findPathExpression(ref *RemoteStateRef) hcl.Expression {
	if expr, ok := ref.Config["key"]; ok {
		return expr
	}
	if expr, ok := ref.Config["prefix"]; ok {
		return expr
	}
	return nil
}

// resolveSimple evaluates a path expression without for_each.
func (r remoteStateResolver) resolveSimple(pathExpr hcl.Expression, evalCtx *hcl.EvalContext) ([]string, error) {
	pathVal, diags := pathExpr.Value(evalCtx)
	if !diags.HasErrors() && pathVal.Type() == cty.String {
		log.WithField("path", pathVal.AsString()).Debug("resolved simple path")
		return []string{pathVal.AsString()}, nil
	}

	log.WithField("reason", "evaluation failed").Debug("falling back to template extraction")
	return extractPathTemplate(pathExpr, evalCtx)
}

// resolveForEach evaluates a path expression for each element in a for_each collection.
func (r remoteStateResolver) resolveForEach(forEachExpr, pathExpr hcl.Expression, evalCtx *hcl.EvalContext) ([]string, error) {
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
			eachKey = v // for sets/lists, key == value
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

// extractPathTemplate attempts to extract a path pattern from an expression.
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
