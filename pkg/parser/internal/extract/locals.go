package extract

import (
	"maps"

	"github.com/hashicorp/hcl/v2"

	"github.com/edelwud/terraci/internal/terraform/eval"
)

func extractLocals(ctx *Context) {
	allAttrs := make(map[string]*hcl.Attribute)
	for _, block := range ctx.Source.LocalsBlocks() {
		attrs, diags := block.Body.JustAttributes()
		ctx.Sink.AddDiags(diags)
		maps.Copy(allAttrs, attrs)
	}

	locals := ctx.Sink.Locals()
	evalCtx := eval.NewContext(locals, ctx.Sink.Variables(), ctx.Sink.Path())
	evalCtx.Variables["local"] = eval.SafeObjectVal(locals)

	const maxPasses = 10
	for range maxPasses {
		resolved := 0
		for name, attr := range allAttrs {
			if _, exists := locals[name]; exists {
				continue
			}

			val, diags := attr.Expr.Value(evalCtx)
			if !diags.HasErrors() && val.IsKnown() {
				ctx.Sink.SetLocal(name, val)
				evalCtx.Variables["local"] = eval.SafeObjectVal(locals)
				resolved++
			}
		}
		if resolved == 0 {
			break
		}
	}
}
