package parser

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/zclconf/go-cty/cty"
	"golang.org/x/sync/errgroup"

	"github.com/edelwud/terraci/internal/discovery"
)

// DependencyExtractor extracts module dependencies from parsed Terraform files
type DependencyExtractor struct {
	parser *Parser
	index  *discovery.ModuleIndex
}

// NewDependencyExtractor creates a new dependency extractor
func NewDependencyExtractor(parser *Parser, index *discovery.ModuleIndex) *DependencyExtractor {
	return &DependencyExtractor{
		parser: parser,
		index:  index,
	}
}

// Dependency represents a dependency between two modules
type Dependency struct {
	// From module (the one that depends)
	From *discovery.Module
	// To module (the dependency)
	To *discovery.Module
	// Type of dependency (e.g., "remote_state", "module_call")
	Type string
	// Name of the remote state data source or module call
	RemoteStateName string
}

// LibraryDependency represents a dependency on a library module
type LibraryDependency struct {
	// ModuleCall is the parsed module call
	ModuleCall *ModuleCall
	// LibraryPath is the resolved absolute path to the library module
	LibraryPath string
}

// ModuleDependencies contains all dependencies for a module
type ModuleDependencies struct {
	Module       *discovery.Module
	Dependencies []*Dependency
	// LibraryDependencies lists library modules this module uses
	LibraryDependencies []*LibraryDependency
	// DependsOn lists module IDs this module depends on
	DependsOn []string
	// Errors encountered during extraction
	Errors []error
}

// ExtractDependencies extracts dependencies for a single module
func (de *DependencyExtractor) ExtractDependencies(module *discovery.Module) (*ModuleDependencies, error) {
	result := &ModuleDependencies{
		Module:              module,
		Dependencies:        make([]*Dependency, 0),
		LibraryDependencies: make([]*LibraryDependency, 0),
		DependsOn:           make([]string, 0),
		Errors:              make([]error, 0),
	}

	// Parse the module
	parsed, err := de.parser.ParseModule(module.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse module %s: %w", module.ID(), err)
	}

	// Process each remote state reference
	for _, rs := range parsed.RemoteStates {
		deps, errs := de.resolveRemoteStateDependency(module, rs, parsed.Locals, parsed.Variables)
		result.Dependencies = append(result.Dependencies, deps...)
		result.Errors = append(result.Errors, errs...)
	}

	// Process module calls (for library module dependencies)
	for _, mc := range parsed.ModuleCalls {
		if mc.IsLocal && mc.ResolvedPath != "" {
			result.LibraryDependencies = append(result.LibraryDependencies, &LibraryDependency{
				ModuleCall:  mc,
				LibraryPath: mc.ResolvedPath,
			})
		}
	}

	// Build unique DependsOn list
	seen := make(map[string]bool)
	for _, dep := range result.Dependencies {
		if dep.To != nil && !seen[dep.To.ID()] {
			seen[dep.To.ID()] = true
			result.DependsOn = append(result.DependsOn, dep.To.ID())
		}
	}

	return result, nil
}

// resolveRemoteStateDependency attempts to resolve a remote state to actual module dependencies
func (de *DependencyExtractor) resolveRemoteStateDependency(
	from *discovery.Module,
	rs *RemoteStateRef,
	locals, variables map[string]cty.Value,
) ([]*Dependency, []error) {
	var deps []*Dependency
	var errs []error

	// Try to resolve workspace paths using locals and variables from tfvars
	paths, err := de.parser.ResolveWorkspacePath(rs, from.RelativePath, locals, variables)
	if err != nil {
		errs = append(errs, fmt.Errorf("could not resolve workspace path for %s.%s: %w",
			from.ID(), rs.Name, err))
		return deps, errs
	}

	// Match paths to modules
	for _, path := range paths {
		// Skip paths that still contain unresolved dynamic patterns
		if containsDynamicPattern(path) {
			errs = append(errs, fmt.Errorf("unresolved dynamic path %q for %s.%s (check tfvars files)",
				path, from.ID(), rs.Name))
			continue
		}

		target := de.matchPathToModule(path, from)
		if target != nil {
			deps = append(deps, &Dependency{
				From:            from,
				To:              target,
				Type:            "remote_state",
				RemoteStateName: rs.Name,
			})
		} else {
			errs = append(errs, fmt.Errorf("could not find module for path %q (from %s.%s)",
				path, from.ID(), rs.Name))
		}
	}

	return deps, errs
}

// containsDynamicPattern checks if path contains unresolved dynamic patterns
func containsDynamicPattern(path string) bool {
	return strings.Contains(path, "${lookup(") ||
		strings.Contains(path, "${each.") ||
		strings.Contains(path, "${var.") ||
		strings.Contains(path, "\"}")
}

// matchPathToModule matches a state file path to a module
func (de *DependencyExtractor) matchPathToModule(statePath string, from *discovery.Module) *discovery.Module {
	// Common patterns for state file paths:
	// - service/environment/region/module/terraform.tfstate
	// - service/environment/region/module/submodule/terraform.tfstate
	// - service/environment/region/module.tfstate
	// - env:/environment/service/region/module/terraform.tfstate

	// Normalize the path
	statePath = strings.TrimSuffix(statePath, "/terraform.tfstate")
	statePath = strings.TrimSuffix(statePath, ".tfstate")
	statePath = strings.TrimPrefix(statePath, "env:/")

	// Try direct match first (works for both modules and submodules)
	if m := de.index.ByID(statePath); m != nil {
		return m
	}

	// Try with different path separators
	normalizedPath := strings.ReplaceAll(statePath, "/", string(filepath.Separator))
	if m := de.index.ByID(normalizedPath); m != nil {
		return m
	}

	// Extract components and try to match
	parts := strings.Split(statePath, "/")

	// Try matching service/env/region/module/submodule pattern (5 parts)
	if len(parts) >= 5 {
		startIdx := len(parts) - 5
		moduleID := strings.Join(parts[startIdx:], "/")
		if m := de.index.ByID(moduleID); m != nil {
			return m
		}
	}

	// Try matching service/env/region/module pattern (4 parts)
	if len(parts) >= 4 {
		startIdx := len(parts) - 4
		moduleID := strings.Join(parts[startIdx:], "/")
		if m := de.index.ByID(moduleID); m != nil {
			return m
		}
	}

	// Try pattern matching with wildcards from current module context
	// If from module is cdp/stage/eu-central-1/eks, and path is vpc,
	// try cdp/stage/eu-central-1/vpc
	if len(parts) == 1 {
		sameContextID := fmt.Sprintf("%s/%s/%s/%s",
			from.Service, from.Environment, from.Region, parts[0])
		if m := de.index.ByID(sameContextID); m != nil {
			return m
		}

		// Also try as submodule of the same parent module
		// e.g., from ec2/rabbitmq looking for "redis" -> try ec2/redis
		if from.IsSubmodule() {
			siblingID := fmt.Sprintf("%s/%s/%s/%s/%s",
				from.Service, from.Environment, from.Region, from.Module, parts[0])
			if m := de.index.ByID(siblingID); m != nil {
				return m
			}
		}
	}

	// Try module/submodule format (e.g., "ec2/rabbitmq")
	if len(parts) == 2 {
		sameContextID := fmt.Sprintf("%s/%s/%s/%s/%s",
			from.Service, from.Environment, from.Region, parts[0], parts[1])
		if m := de.index.ByID(sameContextID); m != nil {
			return m
		}
	}

	return nil
}

// maxConcurrentExtractions is the maximum number of modules to extract dependencies from concurrently
const maxConcurrentExtractions = 20

// ExtractAllDependencies extracts dependencies for all modules in the index
func (de *DependencyExtractor) ExtractAllDependencies() (map[string]*ModuleDependencies, []error) {
	results := make(map[string]*ModuleDependencies)
	var allErrors []error
	var mu sync.Mutex

	// Create an errgroup with a concurrency limit
	var g errgroup.Group
	g.SetLimit(maxConcurrentExtractions)

	for _, module := range de.index.All() {
		g.Go(func() error {
			deps, err := de.ExtractDependencies(module)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				allErrors = append(allErrors, err)
				return nil
			}

			results[module.ID()] = deps
			allErrors = append(allErrors, deps.Errors...)
			return nil
		})
	}

	// Wait for all goroutines to finish (errors already collected in allErrors)
	_ = g.Wait() //nolint:errcheck

	return results, allErrors
}
