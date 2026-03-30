package parser

import (
	"maps"
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/internal/terraform/eval"
)

type evalContextBuilder struct {
	segments []string
}

func newEvalContextBuilder(segments []string) evalContextBuilder {
	return evalContextBuilder{segments: segments}
}

func (p *Parser) evalContextBuilder() evalContextBuilder {
	return newEvalContextBuilder(p.segments)
}

// build creates an HCL evaluation context with path-derived locals.
func (b evalContextBuilder) build(modulePath string, locals, variables map[string]cty.Value) *hcl.EvalContext {
	pathParts := strings.Split(modulePath, "/")
	if len(pathParts) == 1 {
		pathParts = strings.Split(modulePath, string(os.PathSeparator))
	}

	pathLocals := b.extractPathLocals(pathParts)
	evalCtx := eval.NewContext(locals, variables, modulePath)

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
func (b evalContextBuilder) extractPathLocals(pathParts []string) map[string]cty.Value {
	numSegs := len(b.segments)
	pathLocals := make(map[string]cty.Value, numSegs+2)

	for i, segName := range b.segments {
		if i < len(pathParts) {
			pathLocals[segName] = cty.StringVal(pathParts[i])
		}
	}

	var scope string
	if len(pathParts) > numSegs {
		submodule := strings.Join(pathParts[numSegs:], "/")
		pathLocals["submodule"] = cty.StringVal(submodule)
		if numSegs > 0 {
			lastSeg := b.segments[numSegs-1]
			if v, ok := pathLocals[lastSeg]; ok {
				scope = v.AsString()
			}
			pathLocals[lastSeg] = cty.StringVal(submodule)
		}
	} else if numSegs > 0 {
		if v, ok := pathLocals[b.segments[numSegs-1]]; ok {
			scope = v.AsString()
		}
	}
	pathLocals["scope"] = cty.StringVal(scope)

	return pathLocals
}
