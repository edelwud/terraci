package parser

import (
	"maps"

	"github.com/hashicorp/hcl/v2"

	"github.com/edelwud/terraci/internal/terraform/eval"
)

func (p *Parser) extractLocals(index *moduleIndex, pm *ParsedModule) {
	allAttrs := make(map[string]*hcl.Attribute)
	for _, block := range index.localsBlocks() {
		attrs, diags := block.Body.JustAttributes()
		pm.addDiags(diags)
		maps.Copy(allAttrs, attrs)
	}

	evalCtx := eval.NewContext(pm.Locals, pm.Variables, pm.Path)

	const maxPasses = 10
	for range maxPasses {
		resolved := 0
		for name, attr := range allAttrs {
			if _, exists := pm.Locals[name]; exists {
				continue
			}

			evalCtx.Variables["local"] = eval.SafeObjectVal(pm.Locals)

			val, diags := attr.Expr.Value(evalCtx)
			if !diags.HasErrors() && val.IsKnown() {
				pm.Locals[name] = val
				resolved++
			}
		}
		if resolved == 0 {
			break
		}
	}
}
