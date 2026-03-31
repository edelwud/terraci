package ciprovidertest

import (
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/parser"
)

// TestModule creates a discovery module using the shared CI-provider test convention.
func TestModule(service, env, region, module string) *discovery.Module {
	return discovery.TestModule(service, env, region, module)
}

// ModuleDependencies builds parser dependency fixtures keyed by module ID.
func ModuleDependencies(modules []*discovery.Module, deps map[string][]string) map[string]*parser.ModuleDependencies {
	result := make(map[string]*parser.ModuleDependencies, len(modules))
	for _, module := range modules {
		result[module.ID()] = &parser.ModuleDependencies{
			Module:    module,
			DependsOn: deps[module.ID()],
		}
	}
	return result
}

// DependencyGraph builds a graph from module fixtures and dependency IDs.
func DependencyGraph(modules []*discovery.Module, deps map[string][]string) *graph.DependencyGraph {
	return graph.BuildFromDependencies(modules, ModuleDependencies(modules, deps))
}
