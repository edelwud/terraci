package source

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/pkg/parser/internal/exprfast"
)

var variableDefaultSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{{Name: "default"}},
}

var backendSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{{Type: "backend", LabelNames: []string{"type"}}},
}

var requiredProvidersSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{{Type: "required_providers"}},
}

var moduleCallSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "source", Required: true},
		{Name: "version"},
	},
}

var remoteStateSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "backend", Required: true},
		{Name: "for_each"},
		{Name: "workspace"},
		{Name: "config"},
	},
	Blocks: []hcl.BlockHeaderSchema{{Type: "config"}},
}

type VariableBlockView struct {
	block *hcl.Block
}

func (v VariableBlockView) Name() string {
	if len(v.block.Labels) == 0 {
		return ""
	}
	return v.block.Labels[0]
}

func (v VariableBlockView) DefaultValue() (cty.Value, bool, hcl.Diagnostics) {
	content, _, diags := v.block.Body.PartialContent(variableDefaultSchema)
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

type BackendBlockView struct {
	block *hcl.Block
}

func (v BackendBlockView) Type() string {
	if len(v.block.Labels) == 0 {
		return ""
	}
	return v.block.Labels[0]
}

func (v BackendBlockView) Attributes() (map[string]*hcl.Attribute, hcl.Diagnostics) {
	return v.block.Body.JustAttributes()
}

type TerraformBlockView struct {
	block *hcl.Block
}

func (v TerraformBlockView) BackendBlocks() ([]BackendBlockView, hcl.Diagnostics) {
	content, _, diags := v.block.Body.PartialContent(backendSchema)
	if content == nil {
		return nil, diags
	}

	views := make([]BackendBlockView, 0, len(content.Blocks))
	for _, block := range content.Blocks {
		if len(block.Labels) == 0 {
			continue
		}
		views = append(views, BackendBlockView{block: block})
	}
	return views, diags
}

func (v TerraformBlockView) RequiredProviderBlocks() ([]*hcl.Block, hcl.Diagnostics) {
	content, _, diags := v.block.Body.PartialContent(requiredProvidersSchema)
	if content == nil {
		return nil, diags
	}
	return append([]*hcl.Block(nil), content.Blocks...), diags
}

type ModuleBlockView struct {
	block *hcl.Block
}

func (v ModuleBlockView) Name() string {
	if len(v.block.Labels) == 0 {
		return ""
	}
	return v.block.Labels[0]
}

func (v ModuleBlockView) Content() (*hcl.BodyContent, hcl.Diagnostics) {
	content, _, diags := v.block.Body.PartialContent(moduleCallSchema)
	return content, diags
}

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

	for _, item := range objExpr.Items {
		key, ok := exprfast.EvalString(item.KeyExpr, nil)
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

func (i *Index) VariableBlockViews() []VariableBlockView {
	blocks := i.VariableBlocks()
	views := make([]VariableBlockView, 0, len(blocks))
	for _, block := range blocks {
		if len(block.Labels) == 0 {
			continue
		}
		views = append(views, VariableBlockView{block: block})
	}
	return views
}

func (i *Index) TerraformBlockViews() []TerraformBlockView {
	blocks := i.TerraformBlocks()
	views := make([]TerraformBlockView, 0, len(blocks))
	for _, block := range blocks {
		views = append(views, TerraformBlockView{block: block})
	}
	return views
}

func (i *Index) ModuleBlockViews() []ModuleBlockView {
	blocks := i.ModuleBlocks()
	views := make([]ModuleBlockView, 0, len(blocks))
	for _, block := range blocks {
		if len(block.Labels) == 0 {
			continue
		}
		views = append(views, ModuleBlockView{block: block})
	}
	return views
}

func (i *Index) RemoteStateBlockViews() []RemoteStateBlockView {
	blocks := i.DataBlocks()
	views := make([]RemoteStateBlockView, 0, len(blocks))
	for _, block := range blocks {
		if len(block.Labels) < 2 || block.Labels[0] != "terraform_remote_state" {
			continue
		}
		views = append(views, RemoteStateBlockView{block: block})
	}
	return views
}
