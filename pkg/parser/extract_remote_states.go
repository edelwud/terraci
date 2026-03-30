package parser

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

func extractRemoteStates(ctx *extractContext) {
	for _, block := range ctx.index.dataBlocks() {
		if len(block.Labels) < 2 || block.Labels[0] != "terraform_remote_state" {
			continue
		}

		ref := &RemoteStateRef{
			Name:    block.Labels[1],
			Config:  make(map[string]hcl.Expression),
			RawBody: block.Body,
		}
		parseRemoteStateBlock(ctx, ref, block.Body)
		ctx.parsed.RemoteStates = append(ctx.parsed.RemoteStates, ref)
	}
}

func parseRemoteStateBlock(ctx *extractContext, ref *RemoteStateRef, body hcl.Body) {
	content, _, diags := body.PartialContent(remoteStateSchema())
	ctx.addDiags(diags)
	if content == nil {
		return
	}

	if val, ok := evalContentStringAttr(content, "backend"); ok {
		ref.Backend = val
	}
	if attr, ok := content.Attributes["for_each"]; ok {
		ref.ForEach = attr.Expr
	}

	if attr, ok := content.Attributes["config"]; ok {
		if objExpr, isObj := attr.Expr.(*hclsyntax.ObjectConsExpr); isObj {
			for _, item := range objExpr.Items {
				if keyVal, keyDiags := item.KeyExpr.Value(nil); !keyDiags.HasErrors() && keyVal.Type() == cty.String {
					ref.Config[keyVal.AsString()] = item.ValueExpr
				}
			}
		}
	}

	for _, block := range content.Blocks {
		if block.Type == "config" {
			attrs, blockDiags := block.Body.JustAttributes()
			ctx.addDiags(blockDiags)
			for name, attr := range attrs {
				ref.Config[name] = attr.Expr
			}
		}
	}
}
