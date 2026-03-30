package parser

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

func (p *Parser) extractRemoteStates(index *moduleIndex, pm *ParsedModule) {
	for _, block := range index.dataBlocks() {
		if len(block.Labels) < 2 || block.Labels[0] != "terraform_remote_state" {
			continue
		}

		ref := &RemoteStateRef{
			Name:    block.Labels[1],
			Config:  make(map[string]hcl.Expression),
			RawBody: block.Body,
		}
		parseRemoteStateBlock(ref, block.Body, pm)
		pm.RemoteStates = append(pm.RemoteStates, ref)
	}
}

func parseRemoteStateBlock(ref *RemoteStateRef, body hcl.Body, pm *ParsedModule) {
	content, _, diags := body.PartialContent(remoteStateSchema())
	pm.addDiags(diags)
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
			pm.addDiags(blockDiags)
			for name, attr := range attrs {
				ref.Config[name] = attr.Expr
			}
		}
	}
}
