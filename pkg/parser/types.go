// Package parser provides HCL parsing functionality for Terraform files.
package parser

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// ModuleParser is the interface for parsing Terraform modules.
type ModuleParser interface {
	ParseModule(ctx context.Context, modulePath string) (*ParsedModule, error)
	ResolveWorkspacePath(ref *RemoteStateRef, modulePath string, locals, variables map[string]cty.Value) ([]string, error)
}

// Parser handles parsing of Terraform HCL files.
type Parser struct {
	segments []string
}

// NewParser creates a new HCL parser with the given pattern segments.
func NewParser(segments []string) *Parser {
	if len(segments) == 0 {
		segments = []string{"service", "environment", "region", "module"}
	}
	return &Parser{segments: append([]string{}, segments...)}
}

// Segments returns the parser's configured pattern segments.
func (p *Parser) Segments() []string { return p.segments }

// ParsedModule contains the parsed content of a Terraform module.
type ParsedModule struct {
	Path         string
	Locals       map[string]cty.Value
	Variables    map[string]cty.Value
	Backend      *BackendConfig
	RemoteStates []*RemoteStateRef
	ModuleCalls  []*ModuleCall
	Files        map[string]*hcl.File
	Diagnostics  hcl.Diagnostics
}

// ModuleCall represents a module block in Terraform.
type ModuleCall struct {
	Name         string
	Source       string
	Version      string
	IsLocal      bool
	ResolvedPath string
}

// BackendConfig represents a module's terraform backend configuration.
type BackendConfig struct {
	Type   string            // "s3", "gcs", "azurerm"
	Config map[string]string // evaluated string attributes: bucket, key, region, etc.
}

// RemoteStateRef represents a terraform_remote_state data source reference.
type RemoteStateRef struct {
	Name         string
	Backend      string
	Config       map[string]hcl.Expression
	ForEach      hcl.Expression
	WorkspaceDir string
	RawBody      hcl.Body
}
