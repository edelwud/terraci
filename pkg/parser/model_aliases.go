package parser

import parsermodel "github.com/edelwud/terraci/pkg/parser/model"

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
