package parser

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

type variableBlockView struct {
	block *hcl.Block
}

func (v variableBlockView) Name() string {
	if len(v.block.Labels) == 0 {
		return ""
	}
	return v.block.Labels[0]
}

func (v variableBlockView) DefaultValue() (cty.Value, bool, hcl.Diagnostics) {
	content, _, diags := v.block.Body.PartialContent(variableDefaultSchema())
	if content == nil {
		return cty.NilVal, false, diags
	}

	attr, ok := content.Attributes["default"]
	if !ok {
		return cty.NilVal, false, diags
	}

	val, valDiags := attr.Expr.Value(nil)
	diags = append(diags, valDiags...)
	if valDiags.HasErrors() {
		return cty.NilVal, false, diags
	}

	return val, true, diags
}

type backendBlockView struct {
	block *hcl.Block
}

func (v backendBlockView) Type() string {
	if len(v.block.Labels) == 0 {
		return ""
	}
	return v.block.Labels[0]
}

func (v backendBlockView) Attributes() (map[string]*hcl.Attribute, hcl.Diagnostics) {
	attrs, diags := v.block.Body.JustAttributes()
	return attrs, diags
}

type terraformBlockView struct {
	block *hcl.Block
}

func (v terraformBlockView) BackendBlocks() ([]backendBlockView, hcl.Diagnostics) {
	content, _, diags := v.block.Body.PartialContent(backendSchema())
	if content == nil {
		return nil, diags
	}

	views := make([]backendBlockView, 0, len(content.Blocks))
	for _, block := range content.Blocks {
		if len(block.Labels) == 0 {
			continue
		}
		views = append(views, backendBlockView{block: block})
	}

	return views, diags
}

func (v terraformBlockView) RequiredProviderBlocks() ([]*hcl.Block, hcl.Diagnostics) {
	content, _, diags := v.block.Body.PartialContent(requiredProvidersSchema())
	if content == nil {
		return nil, diags
	}

	return append([]*hcl.Block(nil), content.Blocks...), diags
}

type moduleBlockView struct {
	block *hcl.Block
}

func (v moduleBlockView) Name() string {
	if len(v.block.Labels) == 0 {
		return ""
	}
	return v.block.Labels[0]
}

func (v moduleBlockView) Content() (*hcl.BodyContent, hcl.Diagnostics) {
	content, _, diags := v.block.Body.PartialContent(moduleCallSchema())
	return content, diags
}

type remoteStateBlockView struct {
	block *hcl.Block
}

func (v remoteStateBlockView) Name() string {
	if len(v.block.Labels) < 2 {
		return ""
	}
	return v.block.Labels[1]
}

func (v remoteStateBlockView) RawBody() hcl.Body {
	return v.block.Body
}

func (v remoteStateBlockView) Content() (*hcl.BodyContent, hcl.Diagnostics) {
	content, _, diags := v.block.Body.PartialContent(remoteStateSchema())
	return content, diags
}

func (v remoteStateBlockView) InlineConfigExpressions(content *hcl.BodyContent) map[string]hcl.Expression {
	config := make(map[string]hcl.Expression)
	attr, ok := content.Attributes["config"]
	if !ok {
		return config
	}

	objExpr, isObj := attr.Expr.(*hclsyntax.ObjectConsExpr)
	if !isObj {
		return config
	}

	for _, item := range objExpr.Items {
		keyVal, keyDiags := item.KeyExpr.Value(nil)
		if keyDiags.HasErrors() || keyVal.Type() != cty.String {
			continue
		}
		config[keyVal.AsString()] = item.ValueExpr
	}

	return config
}

func (v remoteStateBlockView) ConfigBlockAttributes(content *hcl.BodyContent) (map[string]hcl.Expression, hcl.Diagnostics) {
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

func (i *moduleIndex) variableBlockViews() []variableBlockView {
	blocks := i.variableBlocks()
	views := make([]variableBlockView, 0, len(blocks))
	for _, block := range blocks {
		if len(block.Labels) == 0 {
			continue
		}
		views = append(views, variableBlockView{block: block})
	}
	return views
}

func (i *moduleIndex) terraformBlockViews() []terraformBlockView {
	blocks := i.terraformBlocks()
	views := make([]terraformBlockView, 0, len(blocks))
	for _, block := range blocks {
		views = append(views, terraformBlockView{block: block})
	}
	return views
}

func (i *moduleIndex) moduleBlockViews() []moduleBlockView {
	blocks := i.moduleBlocks()
	views := make([]moduleBlockView, 0, len(blocks))
	for _, block := range blocks {
		if len(block.Labels) == 0 {
			continue
		}
		views = append(views, moduleBlockView{block: block})
	}
	return views
}

func (i *moduleIndex) remoteStateBlockViews() []remoteStateBlockView {
	blocks := i.dataBlocks()
	views := make([]remoteStateBlockView, 0, len(blocks))
	for _, block := range blocks {
		if len(block.Labels) < 2 || block.Labels[0] != "terraform_remote_state" {
			continue
		}
		views = append(views, remoteStateBlockView{block: block})
	}
	return views
}
