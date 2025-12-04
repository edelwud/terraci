// Package parser provides HCL parsing functionality for Terraform files
package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

// Parser handles parsing of Terraform HCL files
type Parser struct {
	hclParser *hclparse.Parser
}

// NewParser creates a new HCL parser
func NewParser() *Parser {
	return &Parser{
		hclParser: hclparse.NewParser(),
	}
}

// ParsedModule contains the parsed content of a Terraform module
type ParsedModule struct {
	// Path to the module directory
	Path string
	// Locals extracted from locals.tf
	Locals map[string]cty.Value
	// RemoteStates extracted from all .tf files
	RemoteStates []*RemoteStateRef
	// Raw HCL files for further analysis
	Files map[string]*hcl.File
	// Diagnostics from parsing
	Diagnostics hcl.Diagnostics
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
		RemoteStates: make([]*RemoteStateRef, 0),
		Files:        make(map[string]*hcl.File),
	}

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

		file, diags := p.hclParser.ParseHCL(content, tfFile)
		result.Diagnostics = append(result.Diagnostics, diags...)

		if file != nil {
			result.Files[tfFile] = file
		}
	}

	// Extract locals
	if err := p.extractLocals(result); err != nil {
		return nil, fmt.Errorf("failed to extract locals: %w", err)
	}

	// Extract remote state references
	if err := p.extractRemoteStates(result); err != nil {
		return nil, fmt.Errorf("failed to extract remote states: %w", err)
	}

	return result, nil
}

// extractLocals parses locals blocks from the module files
func (p *Parser) extractLocals(pm *ParsedModule) error {
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

	return nil
}

// extractRemoteStates parses terraform_remote_state data sources
func (p *Parser) extractRemoteStates(pm *ParsedModule) error {
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

	return nil
}

// parseRemoteStateBlock extracts configuration from a terraform_remote_state block
func (p *Parser) parseRemoteStateBlock(ref *RemoteStateRef, body hcl.Body, pm *ParsedModule) {
	// Schema for terraform_remote_state
	schema := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "backend", Required: true},
			{Name: "for_each"},
			{Name: "workspace"},
		},
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "config"},
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

	// Extract config block
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
// This uses the module's locals and path information to resolve variables
func (p *Parser) ResolveWorkspacePath(ref *RemoteStateRef, modulePath string, locals map[string]cty.Value) ([]string, error) {
	// Build evaluation context with locals and path-derived variables
	pathParts := strings.Split(modulePath, string(os.PathSeparator))

	// Extract path components (assuming service/environment/region/module structure)
	var service, environment, region, module string
	if len(pathParts) >= 4 {
		module = pathParts[len(pathParts)-1]
		region = pathParts[len(pathParts)-2]
		environment = pathParts[len(pathParts)-3]
		service = pathParts[len(pathParts)-4]
	}

	// Create evaluation context
	evalCtx := &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"local": cty.ObjectVal(locals),
			"path": cty.ObjectVal(map[string]cty.Value{
				"module": cty.StringVal(modulePath),
			}),
		},
	}

	// Add path-derived locals if not already present
	pathLocals := map[string]cty.Value{
		"service":     cty.StringVal(service),
		"environment": cty.StringVal(environment),
		"region":      cty.StringVal(region),
		"module":      cty.StringVal(module),
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
			// Can't evaluate for_each statically, return template
			return p.extractPathTemplate(pathExpr, evalCtx)
		}

		// Iterate over for_each values
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
					paths = append(paths, pathVal.AsString())
				}
			}
		}
	} else {
		// Simple case without for_each
		pathVal, diags := pathExpr.Value(evalCtx)
		if !diags.HasErrors() && pathVal.Type() == cty.String {
			paths = append(paths, pathVal.AsString())
		} else {
			// Try to extract template
			return p.extractPathTemplate(pathExpr, evalCtx)
		}
	}

	return paths, nil
}

// extractPathTemplate attempts to extract a path pattern from an expression
func (p *Parser) extractPathTemplate(expr hcl.Expression, ctx *hcl.EvalContext) ([]string, error) {
	// Try partial evaluation
	val, _ := expr.Value(ctx)
	if val.Type() == cty.String {
		return []string{val.AsString()}, nil
	}

	// Return the expression source as a template
	rng := expr.Range()
	if rng.Filename != "" {
		content, err := os.ReadFile(rng.Filename)
		if err == nil {
			start := rng.Start.Byte
			end := rng.End.Byte
			if int(end) <= len(content) {
				template := string(content[start:end])
				return []string{template}, nil
			}
		}
	}

	return nil, fmt.Errorf("could not extract path template")
}
