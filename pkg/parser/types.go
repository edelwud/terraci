// Package parser provides HCL parsing functionality for Terraform files.
package parser

import (
	"context"

	"github.com/zclconf/go-cty/cty"

	parsermodel "github.com/edelwud/terraci/pkg/parser/model"
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
func (p *Parser) Segments() []string { return append([]string(nil), p.segments...) }

// ParsedModule contains the parsed content of a Terraform module.
type ParsedModule = parsermodel.ParsedModule

// NewParsedModule creates an empty parsed module with initialized collections.
func NewParsedModule(modulePath string) *ParsedModule {
	return parsermodel.NewParsedModule(modulePath)
}

// RequiredProvider represents a provider requirement from a required_providers block.
type RequiredProvider = parsermodel.RequiredProvider

// LockedProvider represents a provider entry from .terraform.lock.hcl.
type LockedProvider = parsermodel.LockedProvider

// ModuleCall represents a module block in Terraform.
type ModuleCall = parsermodel.ModuleCall

// BackendConfig represents a module's terraform backend configuration.
type BackendConfig = parsermodel.BackendConfig

// RemoteStateRef represents a terraform_remote_state data source reference.
type RemoteStateRef = parsermodel.RemoteStateRef

// Dependency represents a dependency between two modules.
type Dependency = parsermodel.Dependency

// LibraryDependency represents a dependency on a library module.
type LibraryDependency = parsermodel.LibraryDependency

// ModuleDependencies contains all dependencies for a module.
type ModuleDependencies = parsermodel.ModuleDependencies
