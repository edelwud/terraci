package extract

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

func extractRequiredProviders(ctx *Context) {
	for _, terraformBlock := range ctx.Index.TerraformBlockViews() {
		requiredProviders, diags := terraformBlock.RequiredProviderBlocks()
		ctx.Sink.AddDiags(diags)
		for _, rpBlock := range requiredProviders {
			attrs, attrDiags := rpBlock.Body.JustAttributes()
			ctx.Sink.AddDiags(attrDiags)

			for name, attr := range attrs {
				rp := RequiredProvider{Name: name}

				if val, valDiags := evalAttrValue(attr); !valDiags.HasErrors() && val.Type() == cty.String {
					rp.VersionConstraint = val.AsString()
					ctx.Sink.AppendRequiredProvider(rp)
					continue
				}

				if objExpr, isObj := attr.Expr.(*hclsyntax.ObjectConsExpr); isObj {
					fillRequiredProviderFromObject(&rp, objExpr)
				}

				ctx.Sink.AppendRequiredProvider(rp)
			}
		}
	}
}

func fillRequiredProviderFromObject(rp *RequiredProvider, objExpr *hclsyntax.ObjectConsExpr) {
	values := evalObjectStringAttrs(objExpr, nil)
	if source, ok := values["source"]; ok {
		rp.Source = source
	}
	if version, ok := values["version"]; ok {
		rp.VersionConstraint = version
	}
}
