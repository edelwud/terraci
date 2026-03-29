package parser

import (
	"errors"
	"maps"
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/internal/terraform/eval"
	"github.com/edelwud/terraci/pkg/log"
)

// ResolveWorkspacePath resolves workspace paths from a remote state config
// using the module's locals, variables, and path-derived segment values.
func (p *Parser) ResolveWorkspacePath(ref *RemoteStateRef, modulePath string, locals, variables map[string]cty.Value) ([]string, error) {
	log.WithField("module", modulePath).WithField("remote_state", ref.Name).Debug("resolving workspace path")

	evalCtx := p.buildEvalContext(modulePath, locals, variables)

	pathExpr := p.findPathExpression(ref)
	if pathExpr == nil {
		return nil, errors.New("no key or prefix found in remote state config")
	}

	if ref.ForEach != nil {
		return p.resolveForEach(ref.ForEach, pathExpr, evalCtx)
	}

	return p.resolveSimple(pathExpr, evalCtx)
}

// buildEvalContext creates an HCL evaluation context with path-derived locals.
func (p *Parser) buildEvalContext(modulePath string, locals, variables map[string]cty.Value) *hcl.EvalContext {
	pathParts := strings.Split(modulePath, "/")
	if len(pathParts) == 1 {
		pathParts = strings.Split(modulePath, string(os.PathSeparator))
	}

	pathLocals := p.extractPathLocals(pathParts)

	evalCtx := eval.NewContext(locals, variables, modulePath)

	// Merge path locals with module locals (module locals take precedence)
	merged := make(map[string]cty.Value, len(locals)+len(pathLocals))
	maps.Copy(merged, locals)
	for k, v := range pathLocals {
		if _, exists := merged[k]; !exists {
			merged[k] = v
		}
	}
	evalCtx.Variables["local"] = cty.ObjectVal(merged)

	return evalCtx
}

// extractPathLocals derives locals from the module path based on configured segments.
func (p *Parser) extractPathLocals(pathParts []string) map[string]cty.Value {
	numSegs := len(p.segments)
	pathLocals := make(map[string]cty.Value, numSegs+2)

	// Map path parts to segment names positionally (first N parts → segments)
	for i, segName := range p.segments {
		if i < len(pathParts) {
			pathLocals[segName] = cty.StringVal(pathParts[i])
		}
	}

	// Handle submodule (path deeper than segments)
	// For platform/vpn/eu-north-1/proxy/prod with pattern {service}/{environment}/{region}/{module}:
	//   service=platform, environment=vpn, region=eu-north-1, module=proxy, submodule=prod
	var scope string
	if len(pathParts) > numSegs {
		submodule := strings.Join(pathParts[numSegs:], "/")
		pathLocals["submodule"] = cty.StringVal(submodule)
		// scope = the last segment value (parent module name)
		if numSegs > 0 {
			lastSeg := p.segments[numSegs-1]
			if v, ok := pathLocals[lastSeg]; ok {
				scope = v.AsString()
			}
			// Override last segment's local with the submodule name for Terraform compatibility
			// (local.module in submodule code typically refers to the submodule itself)
			pathLocals[lastSeg] = cty.StringVal(submodule)
		}
	} else if numSegs > 0 {
		if v, ok := pathLocals[p.segments[numSegs-1]]; ok {
			scope = v.AsString()
		}
	}
	pathLocals["scope"] = cty.StringVal(scope)

	return pathLocals
}

// findPathExpression returns the key or prefix expression from remote state config.
func (p *Parser) findPathExpression(ref *RemoteStateRef) hcl.Expression {
	if expr, ok := ref.Config["key"]; ok {
		return expr
	}
	if expr, ok := ref.Config["prefix"]; ok {
		return expr
	}
	return nil
}

// resolveSimple evaluates a path expression without for_each.
func (p *Parser) resolveSimple(pathExpr hcl.Expression, evalCtx *hcl.EvalContext) ([]string, error) {
	pathVal, diags := pathExpr.Value(evalCtx)
	if !diags.HasErrors() && pathVal.Type() == cty.String {
		log.WithField("path", pathVal.AsString()).Debug("resolved simple path")
		return []string{pathVal.AsString()}, nil
	}

	log.WithField("reason", "evaluation failed").Debug("falling back to template extraction")
	return extractPathTemplate(pathExpr, evalCtx)
}

// resolveForEach evaluates a path expression for each element in a for_each collection.
func (p *Parser) resolveForEach(forEachExpr, pathExpr hcl.Expression, evalCtx *hcl.EvalContext) ([]string, error) {
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
