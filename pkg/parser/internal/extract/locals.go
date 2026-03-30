package extract

import (
	"maps"

	"github.com/hashicorp/hcl/v2"

	"github.com/edelwud/terraci/internal/terraform/eval"
)

func extractLocals(ctx *Context) {
	allAttrs := make(map[string]*hcl.Attribute)
	for _, block := range ctx.Index.LocalsBlocks() {
		attrs, diags := block.Body.JustAttributes()
		ctx.Sink.AddDiags(diags)
		maps.Copy(allAttrs, attrs)
	}

	evalCtx := eval.NewContext(ctx.Sink.Locals(), ctx.Sink.Variables(), ctx.Sink.Path())

	const maxPasses = 10
	for range maxPasses {
		resolved := 0
		for name, attr := range allAttrs {
			if _, exists := ctx.Sink.Locals()[name]; exists {
				continue
			}

			evalCtx.Variables["local"] = eval.SafeObjectVal(ctx.Sink.Locals())

			val, diags := attr.Expr.Value(evalCtx)
			if !diags.HasErrors() && val.IsKnown() {
				ctx.Sink.SetLocal(name, val)
				resolved++
			}
		}
		if resolved == 0 {
			break
		}
	}
}
