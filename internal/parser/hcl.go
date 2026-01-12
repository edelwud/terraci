// Package parser provides HCL parsing functionality for Terraform files
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
	"github.com/edelwud/terraci/pkg/log"
)

// Parser handles parsing of Terraform HCL files
type Parser struct {
}

// NewParser creates a new HCL parser
func NewParser() *Parser {
	return &Parser{}
}

// ParsedModule contains the parsed content of a Terraform module
type ParsedModule struct {
	// Path to the module directory
	Path string
	// Locals extracted from locals.tf
	Locals map[string]cty.Value
	// Variables extracted from *.auto.tfvars files
	Variables map[string]cty.Value
	// RemoteStates extracted from all .tf files
	RemoteStates []*RemoteStateRef
	// ModuleCalls extracted from all .tf files (module blocks)
	ModuleCalls []*ModuleCall
	// Raw HCL files for further analysis
	Files map[string]*hcl.File
	// Diagnostics from parsing
	Diagnostics hcl.Diagnostics
}

// ModuleCall represents a module block in Terraform
type ModuleCall struct {
	// Name of the module call (e.g., "vpc" in module "vpc" { ... })
	Name string
	// Source is the module source (e.g., "../../../_modules/kafka", "terraform-aws-modules/vpc/aws")
	Source string
	// Version is the module version constraint (for registry modules)
	Version string
	// IsLocal indicates if the source is a local path
	IsLocal bool
	// ResolvedPath is the absolute path for local modules
	ResolvedPath string
}

// RemoteStateRef represents a terraform_remote_state data source reference
type RemoteStateRef struct {
	// Name of the data source (e.g., "vpc" in data.terraform_remote_state.vpc)
	Name string
	// Backend type (e.g., "s3", "gcs")
	Backend string
	// Config contains the backend configuration
	Config map[string]hcl.Expression
	// ForEach expression if for_each is used
	ForEach hcl.Expression
	// WorkspaceDir is the resolved workspace directory path pattern
	// This is extracted from config.key or config.prefix for S3 backend
	WorkspaceDir string
	// Raw attributes for further processing
	RawBody hcl.Body
}

// ParseModule parses all Terraform files in a module directory
func (p *Parser) ParseModule(modulePath string) (*ParsedModule, error) {
	result := &ParsedModule{
		Path:         modulePath,
		Locals:       make(map[string]cty.Value),
		Variables:    make(map[string]cty.Value),
		RemoteStates: make([]*RemoteStateRef, 0),
		ModuleCalls:  make([]*ModuleCall, 0),
		Files:        make(map[string]*hcl.File),
	}

	// Use a local parser instance for thread safety
	hclParser := hclparse.NewParser()

	// Find all .tf files in the directory
	tfFiles, err := filepath.Glob(filepath.Join(modulePath, "*.tf"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob .tf files: %w", err)
	}

	// Parse each file
	for _, tfFile := range tfFiles {
		content, err := os.ReadFile(tfFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", tfFile, err)
		}

		file, diags := hclParser.ParseHCL(content, tfFile)
		result.Diagnostics = append(result.Diagnostics, diags...)

		if file != nil {
			result.Files[tfFile] = file
		}
	}

	// Extract locals
	p.extractLocals(result)

	// Extract variables from .auto.tfvars files
	p.extractTfvars(result, hclParser)

	// Extract remote state references
	p.extractRemoteStates(result)

	// Extract module calls
	p.extractModuleCalls(result)

	return result, nil
}

// extractLocals parses locals blocks from the module files
func (p *Parser) extractLocals(pm *ParsedModule) {
	localsSchema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "locals"},
		},
	}

	for _, file := range pm.Files {
		content, _, diags := file.Body.PartialContent(localsSchema)
		pm.Diagnostics = append(pm.Diagnostics, diags...)

		if content == nil {
			continue
		}

		for _, block := range content.Blocks {
			if block.Type != "locals" {
				continue
			}

			attrs, diags := block.Body.JustAttributes()
			pm.Diagnostics = append(pm.Diagnostics, diags...)

			for name, attr := range attrs {
				// Try to evaluate simple expressions
				val, diags := attr.Expr.Value(nil)
				if !diags.HasErrors() {
					pm.Locals[name] = val
				}
			}
		}
	}
}

// extractTfvars parses variable definitions and their values from multiple sources:
// 1. Default values from variables.tf (lowest priority)
// 2. terraform.tfvars
// 3. *.auto.tfvars (highest priority)
func (p *Parser) extractTfvars(pm *ParsedModule, hclParser *hclparse.Parser) {
	// First, extract default values from variable blocks in .tf files
	p.extractVariableDefaults(pm)

	// Then load terraform.tfvars (overrides defaults)
	terraformTfvars := filepath.Join(pm.Path, "terraform.tfvars")
	if _, err := os.Stat(terraformTfvars); err == nil {
		p.loadTfvarsFile(pm, terraformTfvars, hclParser)
	}

	// Finally, load *.auto.tfvars files (highest priority)
	tfvarsFiles, err := filepath.Glob(filepath.Join(pm.Path, "*.auto.tfvars"))
	if err != nil {
		return
	}

	for _, tfvarsFile := range tfvarsFiles {
		p.loadTfvarsFile(pm, tfvarsFile, hclParser)
	}
}

// extractVariableDefaults extracts default values from variable blocks
func (p *Parser) extractVariableDefaults(pm *ParsedModule) {
	varSchema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "variable", LabelNames: []string{"name"}},
		},
	}

	for _, file := range pm.Files {
		content, _, diags := file.Body.PartialContent(varSchema)
		pm.Diagnostics = append(pm.Diagnostics, diags...)

		if content == nil {
			continue
		}

		for _, block := range content.Blocks {
			if block.Type != "variable" || len(block.Labels) < 1 {
				continue
			}

			varName := block.Labels[0]

			// Extract the default attribute
			defaultSchema := &hcl.BodySchema{
				Attributes: []hcl.AttributeSchema{
					{Name: "default"},
				},
			}

			blockContent, _, diags := block.Body.PartialContent(defaultSchema)
			pm.Diagnostics = append(pm.Diagnostics, diags...)

			if blockContent == nil {
				continue
			}

			if attr, ok := blockContent.Attributes["default"]; ok {
				val, diags := attr.Expr.Value(nil)
				if !diags.HasErrors() {
					pm.Variables[varName] = val
				}
			}
		}
	}
}

// loadTfvarsFile loads variables from a single tfvars file
func (p *Parser) loadTfvarsFile(pm *ParsedModule, tfvarsFile string, hclParser *hclparse.Parser) {
	content, err := os.ReadFile(tfvarsFile)
	if err != nil {
		pm.Diagnostics = append(pm.Diagnostics, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Failed to read tfvars file",
			Detail:   fmt.Sprintf("Could not read %s: %v", tfvarsFile, err),
		})
		return
	}

	file, diags := hclParser.ParseHCL(content, tfvarsFile)
	pm.Diagnostics = append(pm.Diagnostics, diags...)

	if file == nil {
		return
	}

	// tfvars files are just attribute assignments at the top level
	attrs, diags := file.Body.JustAttributes()
	pm.Diagnostics = append(pm.Diagnostics, diags...)

	for name, attr := range attrs {
		// Try to evaluate the expression
		val, diags := attr.Expr.Value(nil)
		if !diags.HasErrors() {
			pm.Variables[name] = val
		}
	}
}

// extractRemoteStates parses terraform_remote_state data sources
func (p *Parser) extractRemoteStates(pm *ParsedModule) {
	dataSchema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "data", LabelNames: []string{"type", "name"}},
		},
	}

	for _, file := range pm.Files {
		content, _, diags := file.Body.PartialContent(dataSchema)
		pm.Diagnostics = append(pm.Diagnostics, diags...)

		if content == nil {
			continue
		}

		for _, block := range content.Blocks {
			if block.Type != "data" || len(block.Labels) < 2 {
				continue
			}

			if block.Labels[0] != "terraform_remote_state" {
				continue
			}

			ref := &RemoteStateRef{
				Name:    block.Labels[1],
				Config:  make(map[string]hcl.Expression),
				RawBody: block.Body,
			}

			// Parse the block content
			p.parseRemoteStateBlock(ref, block.Body, pm)

			pm.RemoteStates = append(pm.RemoteStates, ref)
		}
	}
}

// extractModuleCalls parses module blocks from the module files
func (p *Parser) extractModuleCalls(pm *ParsedModule) {
	moduleSchema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "module", LabelNames: []string{"name"}},
		},
	}

	for _, file := range pm.Files {
		content, _, diags := file.Body.PartialContent(moduleSchema)
		pm.Diagnostics = append(pm.Diagnostics, diags...)

		if content == nil {
			continue
		}

		for _, block := range content.Blocks {
			if block.Type != "module" || len(block.Labels) < 1 {
				continue
			}

			call := &ModuleCall{
				Name: block.Labels[0],
			}

			// Parse the module block
			p.parseModuleBlock(call, block.Body, pm)

			pm.ModuleCalls = append(pm.ModuleCalls, call)
		}
	}
}

// parseModuleBlock extracts source and version from a module block
func (p *Parser) parseModuleBlock(call *ModuleCall, body hcl.Body, pm *ParsedModule) {
	schema := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "source", Required: true},
			{Name: "version"},
		},
	}

	content, _, diags := body.PartialContent(schema)
	pm.Diagnostics = append(pm.Diagnostics, diags...)

	if content == nil {
		return
	}

	// Extract source
	if attr, ok := content.Attributes["source"]; ok {
		val, diags := attr.Expr.Value(nil)
		if !diags.HasErrors() && val.Type() == cty.String {
			call.Source = val.AsString()

			// Check if it's a local path
			if strings.HasPrefix(call.Source, "./") || strings.HasPrefix(call.Source, "../") {
				call.IsLocal = true
				// Resolve the absolute path
				call.ResolvedPath = filepath.Clean(filepath.Join(pm.Path, call.Source))
			}
		}
	}

	// Extract version
	if attr, ok := content.Attributes["version"]; ok {
		val, diags := attr.Expr.Value(nil)
		if !diags.HasErrors() && val.Type() == cty.String {
			call.Version = val.AsString()
		}
	}
}

// parseRemoteStateBlock extracts configuration from a terraform_remote_state block
func (p *Parser) parseRemoteStateBlock(ref *RemoteStateRef, body hcl.Body, pm *ParsedModule) {
	// Schema for terraform_remote_state
	// Note: config can be either a block or an attribute (object)
	schema := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "backend", Required: true},
			{Name: "for_each"},
			{Name: "workspace"},
			{Name: "config"}, // config as attribute (config = { ... })
		},
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "config"}, // config as block (config { ... })
		},
	}

	content, _, diags := body.PartialContent(schema)
	pm.Diagnostics = append(pm.Diagnostics, diags...)

	if content == nil {
		return
	}

	// Extract backend
	if attr, ok := content.Attributes["backend"]; ok {
		val, diags := attr.Expr.Value(nil)
		if !diags.HasErrors() && val.Type() == cty.String {
			ref.Backend = val.AsString()
		}
	}

	// Extract for_each if present
	if attr, ok := content.Attributes["for_each"]; ok {
		ref.ForEach = attr.Expr
	}

	// Extract config - can be either attribute or block
	// First, try as attribute (config = { ... })
	if attr, ok := content.Attributes["config"]; ok {
		// config is an object expression, we need to extract its attributes
		// The expression is an object constructor, we can get its items
		if objExpr, ok := attr.Expr.(*hclsyntax.ObjectConsExpr); ok {
			for _, item := range objExpr.Items {
				// Get the key as string
				keyVal, diags := item.KeyExpr.Value(nil)
				if !diags.HasErrors() && keyVal.Type() == cty.String {
					ref.Config[keyVal.AsString()] = item.ValueExpr
				}
			}
		}
	}

	// Then, try as block (config { ... })
	for _, block := range content.Blocks {
		if block.Type == "config" {
			attrs, diags := block.Body.JustAttributes()
			pm.Diagnostics = append(pm.Diagnostics, diags...)

			for name, attr := range attrs {
				ref.Config[name] = attr.Expr
			}
		}
	}
}

// ResolveWorkspacePath attempts to resolve the workspace path from remote state config
// This uses the module's locals, variables, and path information to resolve expressions
func (p *Parser) ResolveWorkspacePath(ref *RemoteStateRef, modulePath string, locals, variables map[string]cty.Value) ([]string, error) {
	log.WithField("module", modulePath).WithField("remote_state", ref.Name).Debug("resolving workspace path")

	// Build evaluation context with locals, variables, and path-derived info
	// Handle both "/" and os.PathSeparator for cross-platform compatibility
	pathParts := strings.Split(modulePath, "/")
	if len(pathParts) == 1 {
		pathParts = strings.Split(modulePath, string(os.PathSeparator))
	}

	// Extract path components
	// Support both depth 4 (service/env/region/module) and depth 5 (service/env/region/module/submodule)
	var service, environment, region, module, submodule string
	if len(pathParts) >= 5 {
		// Submodule: service/env/region/module/submodule
		submodule = pathParts[len(pathParts)-1]
		module = pathParts[len(pathParts)-2]
		region = pathParts[len(pathParts)-3]
		environment = pathParts[len(pathParts)-4]
		service = pathParts[len(pathParts)-5]
	} else if len(pathParts) >= 4 {
		// Base module: service/env/region/module
		module = pathParts[len(pathParts)-1]
		region = pathParts[len(pathParts)-2]
		environment = pathParts[len(pathParts)-3]
		service = pathParts[len(pathParts)-4]
	}

	// For submodules, the "scope" local typically refers to the parent module
	scope := module
	if submodule != "" {
		module = submodule // The actual module name is the submodule
	}

	// Create evaluation context with Terraform functions
	evalCtx := eval.NewContext(locals, variables, modulePath)

	// Add path-derived locals if not already present
	pathLocals := map[string]cty.Value{
		"service":     cty.StringVal(service),
		"environment": cty.StringVal(environment),
		"region":      cty.StringVal(region),
		"module":      cty.StringVal(module),
		"scope":       cty.StringVal(scope),
	}

	// Merge with existing locals
	mergedLocals := make(map[string]cty.Value)
	for k, v := range locals {
		mergedLocals[k] = v
	}
	for k, v := range pathLocals {
		if _, exists := mergedLocals[k]; !exists {
			mergedLocals[k] = v
		}
	}
	evalCtx.Variables["local"] = cty.ObjectVal(mergedLocals)

	var paths []string

	// Try to extract the key/prefix from config
	keyExpr, hasKey := ref.Config["key"]
	prefixExpr, hasPrefix := ref.Config["prefix"]

	var pathExpr hcl.Expression
	if hasKey {
		pathExpr = keyExpr
	} else if hasPrefix {
		pathExpr = prefixExpr
	}

	if pathExpr == nil {
		return nil, fmt.Errorf("no key or prefix found in remote state config")
	}

	// Handle for_each case
	if ref.ForEach != nil {
		// Try to evaluate for_each
		forEachVal, diags := ref.ForEach.Value(evalCtx)
		if diags.HasErrors() {
			log.WithField("reason", "for_each evaluation failed").Debug("falling back to template extraction")
			// Can't evaluate for_each statically, return template
			return p.extractPathTemplate(pathExpr, evalCtx)
		}

		// Iterate over for_each values
		//nolint:dupl // Map/object and set/list iterations are intentionally similar but have different key/value semantics
		if forEachVal.Type().IsMapType() || forEachVal.Type().IsObjectType() {
			for it := forEachVal.ElementIterator(); it.Next(); {
				k, v := it.Element()

				// Create context with each.key and each.value
				iterCtx := evalCtx.NewChild()
				iterCtx.Variables = map[string]cty.Value{
					"each": cty.ObjectVal(map[string]cty.Value{
						"key":   k,
						"value": v,
					}),
				}

				pathVal, diags := pathExpr.Value(iterCtx)
				if !diags.HasErrors() && pathVal.Type() == cty.String {
					log.WithField("path", pathVal.AsString()).Debug("resolved for_each path")
					paths = append(paths, pathVal.AsString())
				}
			}
		} else if forEachVal.Type().IsSetType() || forEachVal.Type().IsTupleType() || forEachVal.Type().IsListType() {
			for it := forEachVal.ElementIterator(); it.Next(); {
				_, v := it.Element()

				iterCtx := evalCtx.NewChild()
				iterCtx.Variables = map[string]cty.Value{
					"each": cty.ObjectVal(map[string]cty.Value{
						"key":   v,
						"value": v,
					}),
				}

				pathVal, diags := pathExpr.Value(iterCtx)
				if !diags.HasErrors() && pathVal.Type() == cty.String {
					log.WithField("path", pathVal.AsString()).Debug("resolved for_each path")
					paths = append(paths, pathVal.AsString())
				}
			}
		}
	} else {
		// Simple case without for_each
		pathVal, diags := pathExpr.Value(evalCtx)
		if !diags.HasErrors() && pathVal.Type() == cty.String {
			log.WithField("path", pathVal.AsString()).Debug("resolved simple path")
			paths = append(paths, pathVal.AsString())
		} else {
			log.WithField("reason", "evaluation failed").Debug("falling back to template extraction")
			// Try to extract template
			return p.extractPathTemplate(pathExpr, evalCtx)
		}
	}

	return paths, nil
}

// extractPathTemplate attempts to extract a path pattern from an expression
func (p *Parser) extractPathTemplate(expr hcl.Expression, ctx *hcl.EvalContext) ([]string, error) {
	// Try partial evaluation - ignore diagnostics as we handle unknown values below
	val, diags := expr.Value(ctx)
	_ = diags // Diagnostics expected for partial evaluation with unknown variables
	if val.IsKnown() && val.Type() == cty.String {
		return []string{val.AsString()}, nil
	}

	// Return the expression source as a template
	rng := expr.Range()
	if rng.Filename != "" {
		content, err := os.ReadFile(rng.Filename)
		if err == nil {
			start := rng.Start.Byte
			end := rng.End.Byte
			if end <= len(content) {
				template := string(content[start:end])
				return []string{template}, nil
			}
		}
	}

	return nil, fmt.Errorf("could not extract path template")
}
