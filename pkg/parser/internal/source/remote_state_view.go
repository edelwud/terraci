package source

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/edelwud/terraci/pkg/parser/internal/exprfast"
)

type RemoteStateBlockView struct {
	block *hcl.Block
}

func (v RemoteStateBlockView) Name() string {
	if len(v.block.Labels) < 2 {
		return ""
	}
	return v.block.Labels[1]
}

func (v RemoteStateBlockView) RawBody() hcl.Body {
	return v.block.Body
}

func (v RemoteStateBlockView) Content() (*hcl.BodyContent, hcl.Diagnostics) {
	content, _, diags := v.block.Body.PartialContent(remoteStateSchema)
	return content, diags
}

func (v RemoteStateBlockView) AppendInlineConfigExpressions(
	content *hcl.BodyContent,
	dst map[string]hcl.Expression,
) {
	attr, ok := content.Attributes["config"]
	if !ok {
		return
	}

	objExpr, isObj := attr.Expr.(*hclsyntax.ObjectConsExpr)
	if !isObj {
		return
	}

	evaluator := exprfast.New(nil)
	for _, item := range objExpr.Items {
		key, ok := evaluator.String(item.KeyExpr)
		if !ok {
			continue
		}
		dst[key] = item.ValueExpr
	}
}

func (v RemoteStateBlockView) AppendConfigBlockAttributes(
	content *hcl.BodyContent,
	dst map[string]hcl.Expression,
) hcl.Diagnostics {
	var diags hcl.Diagnostics

	for _, block := range content.Blocks {
		if block.Type != "config" {
			continue
		}
		attrs, blockDiags := block.Body.JustAttributes()
		diags = append(diags, blockDiags...)
		for name, attr := range attrs {
			dst[name] = attr.Expr
		}
	}

	return diags
}
