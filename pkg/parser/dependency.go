package parser

import (
	"context"

	"github.com/edelwud/terraci/pkg/discovery"
	dependencyengine "github.com/edelwud/terraci/pkg/parser/internal/dependency"
	parserdeps "github.com/edelwud/terraci/pkg/parser/internal/deps"
)

// DependencyExtractor extracts module dependencies from parsed Terraform files.
type DependencyExtractor struct {
	parser ModuleParser
	index  *discovery.ModuleIndex
	engine *dependencyengine.Engine
}

// NewDependencyExtractor creates a new dependency extractor.
func NewDependencyExtractor(parser ModuleParser, index *discovery.ModuleIndex) *DependencyExtractor {
	return &DependencyExtractor{
		parser: parser,
		index:  index,
		engine: dependencyengine.NewEngine(newDependencyParserAdapter(parser), index),
	}
}

// Dependency represents a dependency between two modules.
type Dependency struct {
	From            *discovery.Module
	To              *discovery.Module
	Type            string
	RemoteStateName string
}

// LibraryDependency represents a dependency on a library module.
type LibraryDependency struct {
	ModuleCall  *ModuleCall
	LibraryPath string
}

// ModuleDependencies contains all dependencies for a module.
type ModuleDependencies struct {
	Module              *discovery.Module
	Dependencies        []*Dependency
	LibraryDependencies []*LibraryDependency
	DependsOn           []string
	Errors              []error
}

// ExtractDependencies extracts dependencies for a single module.
func (de *DependencyExtractor) ExtractDependencies(ctx context.Context, module *discovery.Module) (*ModuleDependencies, error) {
	deps, err := de.engine.ExtractDependencies(ctx, module)
	if err != nil {
		return nil, err
	}

	return fromDependencyModuleDependencies(deps), nil
}

// matchPathToModule matches a state file path to a module using multiple strategies.
func (de *DependencyExtractor) matchPathToModule(statePath string, from *discovery.Module) *discovery.Module {
	return parserdeps.MatchPathToModule(de.index, statePath, from)
}

// ExtractAllDependencies extracts dependencies for all modules in the index.
func (de *DependencyExtractor) ExtractAllDependencies(ctx context.Context) (map[string]*ModuleDependencies, []error) {
	results, errs := de.engine.ExtractAllDependencies(ctx)
	return fromDependencyModuleDependenciesMap(results), errs
}
