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
		engine: dependencyengine.NewEngine(parser, index),
	}
}

// ExtractDependencies extracts dependencies for a single module.
func (de *DependencyExtractor) ExtractDependencies(ctx context.Context, module *discovery.Module) (*ModuleDependencies, error) {
	return de.engine.ExtractDependencies(ctx, module)
}

// ExtractAllDependencies extracts dependencies for all modules in the index.
func (de *DependencyExtractor) ExtractAllDependencies(ctx context.Context) (map[string]*ModuleDependencies, []error) {
	return de.engine.ExtractAllDependencies(ctx)
}
