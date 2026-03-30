package parser

import (
	"maps"

	"github.com/hashicorp/hcl/v2"

	"github.com/edelwud/terraci/internal/terraform/eval"
)

func extractLocals(ctx *extractContext) {
	allAttrs := make(map[string]*hcl.Attribute)
	for _, block := range ctx.index.localsBlocks() {
		attrs, diags := block.Body.JustAttributes()
		ctx.addDiags(diags)
		maps.Copy(allAttrs, attrs)
	}

	evalCtx := eval.NewContext(ctx.parsed.Locals, ctx.parsed.Variables, ctx.parsed.Path)

	const maxPasses = 10
	for range maxPasses {
		resolved := 0
		for name, attr := range allAttrs {
			if _, exists := ctx.parsed.Locals[name]; exists {
				continue
			}

			evalCtx.Variables["local"] = eval.SafeObjectVal(ctx.parsed.Locals)

			val, diags := attr.Expr.Value(evalCtx)
			if !diags.HasErrors() && val.IsKnown() {
				ctx.parsed.Locals[name] = val
				resolved++
			}
		}
		if resolved == 0 {
			break
		}
	}
}
