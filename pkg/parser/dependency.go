package parser

import (
	"context"

	"github.com/edelwud/terraci/pkg/discovery"
	parserdeps "github.com/edelwud/terraci/pkg/parser/internal/deps"
)

// DependencyExtractor extracts module dependencies from parsed Terraform files.
type DependencyExtractor struct {
	parser       ModuleParser
	index        *discovery.ModuleIndex
	cache        *parsedModuleCache
	backendIndex *backendModuleIndex
}

// NewDependencyExtractor creates a new dependency extractor.
func NewDependencyExtractor(parser ModuleParser, index *discovery.ModuleIndex) *DependencyExtractor {
	return &DependencyExtractor{
		parser:       parser,
		index:        index,
		cache:        newParsedModuleCache(parser),
		backendIndex: newBackendModuleIndex(),
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
	return newDependencySession(ctx, de, module).Run()
}

// matchPathToModule matches a state file path to a module using multiple strategies.
func (de *DependencyExtractor) matchPathToModule(statePath string, from *discovery.Module) *discovery.Module {
	return parserdeps.MatchPathToModule(de.index, statePath, from)
}

func containsDynamicPattern(path string) bool {
	return parserdeps.ContainsDynamicPattern(path)
}

func (de *DependencyExtractor) buildBackendIndex(ctx context.Context) {
	de.backendIndex.Build(ctx, de.index, de.cache)
}

// backendIndexKey builds a lookup key from a module's backend config.
func backendIndexKey(bc *BackendConfig, modulePath string) string {
	return parserdeps.BackendIndexKey(bc.Type, bc.Config["bucket"], bc.Config["key"], modulePath)
}

// maxConcurrentExtractions is the maximum number of concurrent module extractions.
const maxConcurrentExtractions = 20

// ExtractAllDependencies extracts dependencies for all modules in the index.
func (de *DependencyExtractor) ExtractAllDependencies(ctx context.Context) (map[string]*ModuleDependencies, []error) {
	de.buildBackendIndex(ctx)

	results := make(map[string]*ModuleDependencies)
	var allErrors []error
	var collector resultCollector

	collector.run(de.index.All(), func(module *discovery.Module) (*ModuleDependencies, error) {
		return de.ExtractDependencies(ctx, module)
	}, func(module *discovery.Module, deps *ModuleDependencies, err error) {
		if err != nil {
			allErrors = append(allErrors, err)
			return
		}
		results[module.ID()] = deps
		allErrors = append(allErrors, deps.Errors...)
	})

	return results, allErrors
}
