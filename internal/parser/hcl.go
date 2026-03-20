package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/internal/terraform/eval"
)

// ParseModule parses all Terraform files in a module directory.
func (p *Parser) ParseModule(modulePath string) (*ParsedModule, error) {
	pm := &ParsedModule{
		Path:         modulePath,
		Locals:       make(map[string]cty.Value),
		Variables:    make(map[string]cty.Value),
		RemoteStates: make([]*RemoteStateRef, 0),
		ModuleCalls:  make([]*ModuleCall, 0),
		Files:        make(map[string]*hcl.File),
	}

	hclParser := hclparse.NewParser()

	tfFiles, err := filepath.Glob(filepath.Join(modulePath, "*.tf"))
	if err != nil {
		return nil, fmt.Errorf("glob .tf files: %w", err)
	}

	for _, tfFile := range tfFiles {
		content, readErr := os.ReadFile(tfFile)
		if readErr != nil {
			return nil, fmt.Errorf("read %s: %w", tfFile, readErr)
		}
		file, diags := hclParser.ParseHCL(content, tfFile)
		pm.addDiags(diags)
		if file != nil {
			pm.Files[tfFile] = file
		}
	}

	p.extractLocals(pm)
	p.extractTfvars(pm, hclParser)
	p.extractRemoteStates(pm)
	p.extractModuleCalls(pm)

	return pm, nil
}

// --- Locals ---

func (p *Parser) extractLocals(pm *ParsedModule) {
	// Collect all local attributes from all locals blocks
	allAttrs := make(map[string]*hcl.Attribute)
	for _, block := range p.findBlocks(pm, "locals", nil) {
		attrs, diags := block.Body.JustAttributes()
		pm.addDiags(diags)
		for name, attr := range attrs {
			allAttrs[name] = attr
		}
	}

	// Multi-pass evaluation: locals can reference other locals, path.module, functions
	// Each pass tries to evaluate unresolved locals using already-resolved ones
	evalCtx := eval.NewContext(pm.Locals, pm.Variables, pm.Path)

	const maxPasses = 10
	for range maxPasses {
		resolved := 0
		for name, attr := range allAttrs {
			if _, exists := pm.Locals[name]; exists {
				continue // already resolved
			}

			// Update eval context with currently known locals
			evalCtx.Variables["local"] = eval.SafeObjectVal(pm.Locals)

			val, diags := attr.Expr.Value(evalCtx)
			if !diags.HasErrors() && val.IsKnown() {
				pm.Locals[name] = val
				resolved++
			}
		}
		if resolved == 0 {
			break // no progress, remaining locals have unresolvable deps
		}
	}
}

// --- Variables ---

func (p *Parser) extractTfvars(pm *ParsedModule, hclParser *hclparse.Parser) {
	p.extractVariableDefaults(pm)

	if tfvars := filepath.Join(pm.Path, "terraform.tfvars"); fileExists(tfvars) {
		p.loadTfvarsFile(pm, tfvars, hclParser)
	}

	autoFiles, _ := filepath.Glob(filepath.Join(pm.Path, "*.auto.tfvars")) //nolint:errcheck
	for _, f := range autoFiles {
		p.loadTfvarsFile(pm, f, hclParser)
	}
}

func (p *Parser) extractVariableDefaults(pm *ParsedModule) {
	for _, block := range p.findBlocks(pm, "variable", []string{"name"}) {
		if len(block.Labels) < 1 {
			continue
		}

		schema := &hcl.BodySchema{
			Attributes: []hcl.AttributeSchema{{Name: "default"}},
		}
		content, _, diags := block.Body.PartialContent(schema)
		pm.addDiags(diags)
		if content == nil {
			continue
		}
		if attr, ok := content.Attributes["default"]; ok {
			if val, valDiags := attr.Expr.Value(nil); !valDiags.HasErrors() {
				pm.Variables[block.Labels[0]] = val
			}
		}
	}
}

func (p *Parser) loadTfvarsFile(pm *ParsedModule, path string, hclParser *hclparse.Parser) {
	content, err := os.ReadFile(path)
	if err != nil {
		pm.addDiags(hcl.Diagnostics{&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Failed to read tfvars file",
			Detail:   fmt.Sprintf("Could not read %s: %v", path, err),
		}})
		return
	}

	file, diags := hclParser.ParseHCL(content, path)
	pm.addDiags(diags)
	if file == nil {
		return
	}

	attrs, diags := file.Body.JustAttributes()
	pm.addDiags(diags)

	for name, attr := range attrs {
		if val, diags := attr.Expr.Value(nil); !diags.HasErrors() {
			pm.Variables[name] = val
		}
	}
}

// --- Remote states ---

func (p *Parser) extractRemoteStates(pm *ParsedModule) {
	for _, block := range p.findBlocks(pm, "data", []string{"type", "name"}) {
		if len(block.Labels) < 2 || block.Labels[0] != "terraform_remote_state" {
			continue
		}

		ref := &RemoteStateRef{
			Name:    block.Labels[1],
			Config:  make(map[string]hcl.Expression),
			RawBody: block.Body,
		}
		p.parseRemoteStateBlock(ref, block.Body, pm)
		pm.RemoteStates = append(pm.RemoteStates, ref)
	}
}

func (p *Parser) parseRemoteStateBlock(ref *RemoteStateRef, body hcl.Body, pm *ParsedModule) {
	schema := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "backend", Required: true},
			{Name: "for_each"},
			{Name: "workspace"},
			{Name: "config"},
		},
		Blocks: []hcl.BlockHeaderSchema{{Type: "config"}},
	}

	content, _, diags := body.PartialContent(schema)
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

	// config as attribute: config = { key = "...", ... }
	if attr, ok := content.Attributes["config"]; ok {
		if objExpr, isObj := attr.Expr.(*hclsyntax.ObjectConsExpr); isObj {
			for _, item := range objExpr.Items {
				if keyVal, keyDiags := item.KeyExpr.Value(nil); !keyDiags.HasErrors() && keyVal.Type() == cty.String {
					ref.Config[keyVal.AsString()] = item.ValueExpr
				}
			}
		}
	}

	// config as block: config { key = "..." }
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

// --- Module calls ---

func (p *Parser) extractModuleCalls(pm *ParsedModule) {
	for _, block := range p.findBlocks(pm, "module", []string{"name"}) {
		if len(block.Labels) < 1 {
			continue
		}
		call := &ModuleCall{Name: block.Labels[0]}
		p.parseModuleBlock(call, block.Body, pm)
		pm.ModuleCalls = append(pm.ModuleCalls, call)
	}
}

func (p *Parser) parseModuleBlock(call *ModuleCall, body hcl.Body, pm *ParsedModule) {
	schema := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "source", Required: true},
			{Name: "version"},
		},
	}

	content, _, diags := body.PartialContent(schema)
	pm.addDiags(diags)
	if content == nil {
		return
	}

	if src, ok := evalContentStringAttr(content, "source"); ok {
		call.Source = src
		if strings.HasPrefix(src, "./") || strings.HasPrefix(src, "../") {
			call.IsLocal = true
			call.ResolvedPath = filepath.Clean(filepath.Join(pm.Path, src))
		}
	}

	if ver, ok := evalContentStringAttr(content, "version"); ok {
		call.Version = ver
	}
}

// --- Helpers ---

// addDiags appends diagnostics to the parsed module.
func (pm *ParsedModule) addDiags(diags hcl.Diagnostics) {
	pm.Diagnostics = append(pm.Diagnostics, diags...)
}

// findBlocks extracts blocks of the given type from all parsed files.
func (p *Parser) findBlocks(pm *ParsedModule, blockType string, labels []string) []*hcl.Block {
	schema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{{Type: blockType, LabelNames: labels}},
	}

	var blocks []*hcl.Block
	for _, file := range pm.Files {
		content, _, diags := file.Body.PartialContent(schema)
		pm.addDiags(diags)
		if content != nil {
			blocks = append(blocks, content.Blocks...)
		}
	}
	return blocks
}

// evalContentStringAttr evaluates a named attribute from HCL content as a string.
func evalContentStringAttr(content *hcl.BodyContent, name string) (string, bool) {
	attr, ok := content.Attributes[name]
	if !ok {
		return "", false
	}
	return evalStringExpr(attr.Expr, nil)
}

// evalStringExpr evaluates an expression as a string value.
func evalStringExpr(expr hcl.Expression, ctx *hcl.EvalContext) (string, bool) {
	val, diags := expr.Value(ctx)
	if diags.HasErrors() || val.Type() != cty.String {
		return "", false
	}
	return val.AsString(), true
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
