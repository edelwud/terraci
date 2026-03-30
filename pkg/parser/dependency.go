package parser

import (
	"context"

	"github.com/edelwud/terraci/pkg/discovery"
	dependencyengine "github.com/edelwud/terraci/pkg/parser/internal/dependency"
)

// DependencyExtractor extracts module dependencies from parsed Terraform files.
type DependencyExtractor struct {
	engine *dependencyengine.Engine
}

// NewDependencyExtractor creates a new dependency extractor.
func NewDependencyExtractor(parser ModuleParser, index *discovery.ModuleIndex) *DependencyExtractor {
	return &DependencyExtractor{
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

// ExtractAllDependencies extracts dependencies for all modules in the index.
func (de *DependencyExtractor) ExtractAllDependencies(ctx context.Context) (map[string]*ModuleDependencies, []error) {
	results, errs := de.engine.ExtractAllDependencies(ctx)
	return fromDependencyModuleDependenciesMap(results), errs
}
