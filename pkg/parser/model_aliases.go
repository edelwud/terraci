package parser

import parsermodel "github.com/edelwud/terraci/pkg/parser/model"

// Re-exports of the shared parser model under the public parser package.
// Aliases are kept only for types with at least one external caller; the
// authoritative declarations live in pkg/parser/model.
type (
	ParsedModule       = parsermodel.ParsedModule
	RequiredProvider   = parsermodel.RequiredProvider
	LockedProvider     = parsermodel.LockedProvider
	ModuleCall         = parsermodel.ModuleCall
	RemoteStateRef     = parsermodel.RemoteStateRef
	LibraryDependency  = parsermodel.LibraryDependency
	ModuleDependencies = parsermodel.ModuleDependencies
)
