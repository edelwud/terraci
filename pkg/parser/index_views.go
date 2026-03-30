package parser

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/pkg/parser/internal/source"
)

type variableBlockView struct {
	inner source.VariableBlockView
}

func (v variableBlockView) Name() string {
	return v.inner.Name()
}

func (v variableBlockView) DefaultValue() (cty.Value, bool, hcl.Diagnostics) {
	return v.inner.DefaultValue()
}

type backendBlockView struct {
	inner source.BackendBlockView
}

func (v backendBlockView) Type() string {
	return v.inner.Type()
}

func (v backendBlockView) Attributes() (map[string]*hcl.Attribute, hcl.Diagnostics) {
	return v.inner.Attributes()
}

type terraformBlockView struct {
	inner source.TerraformBlockView
}

func (v terraformBlockView) BackendBlocks() ([]backendBlockView, hcl.Diagnostics) {
	blocks, diags := v.inner.BackendBlocks()
	views := make([]backendBlockView, 0, len(blocks))
	for _, block := range blocks {
		views = append(views, backendBlockView{inner: block})
	}
	return views, diags
}

func (v terraformBlockView) RequiredProviderBlocks() ([]*hcl.Block, hcl.Diagnostics) {
	return v.inner.RequiredProviderBlocks()
}

type moduleBlockView struct {
	inner source.ModuleBlockView
}

func (v moduleBlockView) Name() string {
	return v.inner.Name()
}

func (v moduleBlockView) Content() (*hcl.BodyContent, hcl.Diagnostics) {
	return v.inner.Content()
}

type remoteStateBlockView struct {
	inner source.RemoteStateBlockView
}

func (v remoteStateBlockView) Name() string {
	return v.inner.Name()
}

func (v remoteStateBlockView) RawBody() hcl.Body {
	return v.inner.RawBody()
}

func (v remoteStateBlockView) Content() (*hcl.BodyContent, hcl.Diagnostics) {
	return v.inner.Content()
}

func (v remoteStateBlockView) InlineConfigExpressions(content *hcl.BodyContent) map[string]hcl.Expression {
	return v.inner.InlineConfigExpressions(content)
}

func (v remoteStateBlockView) ConfigBlockAttributes(content *hcl.BodyContent) (map[string]hcl.Expression, hcl.Diagnostics) {
	return v.inner.ConfigBlockAttributes(content)
}

func (i *moduleIndex) variableBlockViews() []variableBlockView {
	views := i.inner.VariableBlockViews()
	result := make([]variableBlockView, 0, len(views))
	for _, view := range views {
		result = append(result, variableBlockView{inner: view})
	}
	return result
}

func (i *moduleIndex) terraformBlockViews() []terraformBlockView {
	views := i.inner.TerraformBlockViews()
	result := make([]terraformBlockView, 0, len(views))
	for _, view := range views {
		result = append(result, terraformBlockView{inner: view})
	}
	return result
}

func (i *moduleIndex) moduleBlockViews() []moduleBlockView {
	views := i.inner.ModuleBlockViews()
	result := make([]moduleBlockView, 0, len(views))
	for _, view := range views {
		result = append(result, moduleBlockView{inner: view})
	}
	return result
}

func (i *moduleIndex) remoteStateBlockViews() []remoteStateBlockView {
	views := i.inner.RemoteStateBlockViews()
	result := make([]remoteStateBlockView, 0, len(views))
	for _, view := range views {
		result = append(result, remoteStateBlockView{inner: view})
	}
	return result
}
