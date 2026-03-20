// Package parser provides HCL parsing functionality for Terraform files.
package parser

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// Parser handles parsing of Terraform HCL files.
type Parser struct {
	Segments []string // ordered pattern segment names
}

// NewParser creates a new HCL parser with default segments.
func NewParser() *Parser {
	return &Parser{
		Segments: []string{"service", "environment", "region", "module"},
	}
}

// ParsedModule contains the parsed content of a Terraform module.
type ParsedModule struct {
	Path         string
	Locals       map[string]cty.Value
	Variables    map[string]cty.Value
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

// RemoteStateRef represents a terraform_remote_state data source reference.
type RemoteStateRef struct {
	Name         string
	Backend      string
	Config       map[string]hcl.Expression
	ForEach      hcl.Expression
	WorkspaceDir string
	RawBody      hcl.Body
}
