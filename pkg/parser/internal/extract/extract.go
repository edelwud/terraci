package extract

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/internal/terraform/eval"
	"github.com/edelwud/terraci/pkg/parser/internal/evalctx"
	"github.com/edelwud/terraci/pkg/parser/internal/exprfast"
	"github.com/edelwud/terraci/pkg/parser/internal/source"
)

const lockFileName = ".terraform.lock.hcl"

type Sink interface {
	Path() string
	Locals() map[string]cty.Value
	Variables() map[string]cty.Value
	AddDiags(hcl.Diagnostics)
	SetLocal(name string, value cty.Value)
	SetVariable(name string, value cty.Value)
	SetBackend(Backend)
	AppendRequiredProvider(RequiredProvider)
	AppendLockedProvider(LockedProvider)
	AppendRemoteState(RemoteState)
	AppendModuleCall(ModuleCall)
}

type Context struct {
	Index       *source.Index
	EvalBuilder evalctx.Builder
	Sink        Sink
}

type Backend struct {
	Type   string
	Config map[string]string
}

type RequiredProvider struct {
	Name              string
	Source            string
	VersionConstraint string
}

type LockedProvider struct {
	Source      string
	Version     string
	Constraints string
}

type RemoteState struct {
	Name    string
	Backend string
	Config  map[string]hcl.Expression
	ForEach hcl.Expression
	RawBody hcl.Body
}

type ModuleCall struct {
	Name         string
	Source       string
	Version      string
	IsLocal      bool
	ResolvedPath string
}

func RunDefault(ctx *Context) {
	extractLocals(ctx)
	extractTfvars(ctx)
	extractBackendConfig(ctx)
	extractRequiredProviders(ctx)
	extractLockFile(ctx)
	extractRemoteStates(ctx)
	extractModuleCalls(ctx)
}

func (c *Context) buildEvalContext() *hcl.EvalContext {
	return c.EvalBuilder.Build(c.Sink.Path(), c.Sink.Locals(), c.Sink.Variables())
}

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

func extractTfvars(ctx *Context) {
	extractVariableDefaults(ctx)

	if tfvars := filepath.Join(ctx.Sink.Path(), "terraform.tfvars"); fileExists(tfvars) {
		loadTfvarsFile(ctx, tfvars)
	}

	autoFiles, _ := filepath.Glob(filepath.Join(ctx.Sink.Path(), "*.auto.tfvars")) //nolint:errcheck
	for _, path := range autoFiles {
		loadTfvarsFile(ctx, path)
	}
}

func extractVariableDefaults(ctx *Context) {
	for _, variable := range ctx.Index.VariableBlockViews() {
		val, ok, diags := variable.DefaultValue()
		ctx.Sink.AddDiags(diags)
		if !ok {
			continue
		}
		ctx.Sink.SetVariable(variable.Name(), val)
	}
}

func loadTfvarsFile(ctx *Context, path string) {
	file, err := ctx.Index.ParseHCLFile(path)
	if err != nil {
		ctx.Sink.AddDiags(tfvarsReadDiagnostic(path, err))
		return
	}
	if file == nil {
		return
	}

	attrs, diags := file.Body.JustAttributes()
	ctx.Sink.AddDiags(diags)
	for name, attr := range attrs {
		if val, valDiags := attr.Expr.Value(nil); !valDiags.HasErrors() {
			ctx.Sink.SetVariable(name, val)
		}
	}
}

func extractBackendConfig(ctx *Context) {
	for _, terraformBlock := range ctx.Index.TerraformBlockViews() {
		backends, diags := terraformBlock.BackendBlocks()
		ctx.Sink.AddDiags(diags)
		for _, backend := range backends {
			cfg := extractBackendAttributes(ctx, backend, ctx.buildEvalContext())
			ctx.Sink.SetBackend(Backend{
				Type:   backend.Type(),
				Config: cfg,
			})
			return
		}
	}
}

func extractBackendAttributes(ctx *Context, block source.BackendBlockView, evalCtx *hcl.EvalContext) map[string]string {
	cfg := make(map[string]string)
	attrs, diags := block.Attributes()
	ctx.Sink.AddDiags(diags)
	for name, attr := range attrs {
		if val, ok := exprfast.EvalString(attr.Expr, evalCtx); ok {
			cfg[name] = val
		}
	}
	return cfg
}

func extractLockFile(ctx *Context) {
	lockPath := filepath.Join(ctx.Sink.Path(), lockFileName)
	file, err := ctx.Index.ParseHCLFile(lockPath)
	if err != nil || file == nil {
		return
	}

	bodyContent, _, diags := file.Body.PartialContent(lockFileSchema())
	ctx.Sink.AddDiags(diags)
	if bodyContent == nil {
		return
	}

	for _, block := range bodyContent.Blocks {
		if len(block.Labels) < 1 {
			continue
		}

		lp := LockedProvider{Source: block.Labels[0]}
		attrContent, _, attrDiags := block.Body.PartialContent(lockProviderAttrSchema())
		ctx.Sink.AddDiags(attrDiags)
		if attrContent == nil {
			continue
		}

		if v, ok := exprfast.ContentStringAttr(attrContent, "version", nil); ok {
			lp.Version = v
		}
		if v, ok := exprfast.ContentStringAttr(attrContent, "constraints", nil); ok {
			lp.Constraints = v
		}

		ctx.Sink.AppendLockedProvider(lp)
	}
}

func extractRequiredProviders(ctx *Context) {
	for _, terraformBlock := range ctx.Index.TerraformBlockViews() {
		requiredProviders, diags := terraformBlock.RequiredProviderBlocks()
		ctx.Sink.AddDiags(diags)
		for _, rpBlock := range requiredProviders {
			attrs, attrDiags := rpBlock.Body.JustAttributes()
			ctx.Sink.AddDiags(attrDiags)

			for name, attr := range attrs {
				rp := RequiredProvider{Name: name}

				if val, valDiags := attr.Expr.Value(nil); !valDiags.HasErrors() && val.Type() == cty.String {
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
	for _, item := range objExpr.Items {
		keyVal, keyDiags := item.KeyExpr.Value(nil)
		if keyDiags.HasErrors() || keyVal.Type() != cty.String {
			continue
		}
		switch keyVal.AsString() {
		case "source":
			if v, ok := exprfast.EvalString(item.ValueExpr, nil); ok {
				rp.Source = v
			}
		case "version":
			if v, ok := exprfast.EvalString(item.ValueExpr, nil); ok {
				rp.VersionConstraint = v
			}
		}
	}
}

func extractRemoteStates(ctx *Context) {
	for _, remoteState := range ctx.Index.RemoteStateBlockViews() {
		ref := RemoteState{
			Name:    remoteState.Name(),
			Config:  make(map[string]hcl.Expression),
			RawBody: remoteState.RawBody(),
		}
		parseRemoteStateBlock(ctx, remoteState, &ref)
		ctx.Sink.AppendRemoteState(ref)
	}
}

func parseRemoteStateBlock(ctx *Context, view source.RemoteStateBlockView, ref *RemoteState) {
	content, diags := view.Content()
	ctx.Sink.AddDiags(diags)
	if content == nil {
		return
	}

	if val, ok := exprfast.ContentStringAttr(content, "backend", nil); ok {
		ref.Backend = val
	}
	if attr, ok := content.Attributes["for_each"]; ok {
		ref.ForEach = attr.Expr
	}

	if _, ok := content.Attributes["config"]; ok {
		maps.Copy(ref.Config, view.InlineConfigExpressions(content))
	}

	configAttrs, configDiags := view.ConfigBlockAttributes(content)
	ctx.Sink.AddDiags(configDiags)
	maps.Copy(ref.Config, configAttrs)
}

func extractModuleCalls(ctx *Context) {
	for _, module := range ctx.Index.ModuleBlockViews() {
		call := ModuleCall{Name: module.Name()}
		parseModuleBlock(ctx, module, &call)
		ctx.Sink.AppendModuleCall(call)
	}
}

func parseModuleBlock(ctx *Context, view source.ModuleBlockView, call *ModuleCall) {
	content, diags := view.Content()
	ctx.Sink.AddDiags(diags)
	if content == nil {
		return
	}

	if src, ok := exprfast.ContentStringAttr(content, "source", nil); ok {
		call.Source = src
		if strings.HasPrefix(src, "./") || strings.HasPrefix(src, "../") {
			call.IsLocal = true
			call.ResolvedPath = filepath.Clean(filepath.Join(ctx.Sink.Path(), src))
		}
	}

	if ver, ok := exprfast.ContentStringAttr(content, "version", nil); ok {
		call.Version = ver
	}
}

func lockFileSchema() *hcl.BodySchema {
	return &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{{Type: "provider", LabelNames: []string{"source"}}},
	}
}

func lockProviderAttrSchema() *hcl.BodySchema {
	return &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "version"},
			{Name: "constraints"},
		},
	}
}

func tfvarsReadDiagnostic(path string, err error) hcl.Diagnostics {
	return hcl.Diagnostics{&hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "Failed to read tfvars file",
		Detail:   fmt.Sprintf("Could not read %s: %v", path, err),
	}}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
