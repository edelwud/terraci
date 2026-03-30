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

func (v RemoteStateBlockView) InlineConfigExpressions(content *hcl.BodyContent) map[string]hcl.Expression {
	config := make(map[string]hcl.Expression)
	attr, ok := content.Attributes["config"]
	if !ok {
		return config
	}

	objExpr, isObj := attr.Expr.(*hclsyntax.ObjectConsExpr)
	if !isObj {
		return config
	}

	evaluator := exprfast.New(nil)
	for _, item := range objExpr.Items {
		key, ok := evaluator.String(item.KeyExpr)
		if !ok {
			continue
		}
		config[key] = item.ValueExpr
	}
	return config
}

func (v RemoteStateBlockView) ConfigBlockAttributes(content *hcl.BodyContent) (map[string]hcl.Expression, hcl.Diagnostics) {
	config := make(map[string]hcl.Expression)
	var diags hcl.Diagnostics

	for _, block := range content.Blocks {
		if block.Type != "config" {
			continue
		}
		attrs, blockDiags := block.Body.JustAttributes()
		diags = append(diags, blockDiags...)
		for name, attr := range attrs {
			config[name] = attr.Expr
		}
	}

	return config, diags
}
