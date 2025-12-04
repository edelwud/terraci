package parser

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/terraci/terraci/internal/discovery"
	"github.com/zclconf/go-cty/cty"
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
	// Type of dependency (e.g., "remote_state")
	Type string
	// Name of the remote state data source
	RemoteStateName string
}

// ModuleDependencies contains all dependencies for a module
type ModuleDependencies struct {
	Module       *discovery.Module
	Dependencies []*Dependency
	// DependsOn lists module IDs this module depends on
	DependsOn []string
	// Errors encountered during extraction
	Errors []error
}

// ExtractDependencies extracts dependencies for a single module
func (de *DependencyExtractor) ExtractDependencies(module *discovery.Module) (*ModuleDependencies, error) {
	result := &ModuleDependencies{
		Module:       module,
		Dependencies: make([]*Dependency, 0),
		DependsOn:    make([]string, 0),
		Errors:       make([]error, 0),
	}

	// Parse the module
	parsed, err := de.parser.ParseModule(module.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse module %s: %w", module.ID(), err)
	}

	// Process each remote state reference
	for _, rs := range parsed.RemoteStates {
		deps, errs := de.resolveRemoteStateDependency(module, rs, parsed.Locals)
		result.Dependencies = append(result.Dependencies, deps...)
		result.Errors = append(result.Errors, errs...)
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
	locals map[string]cty.Value,
) ([]*Dependency, []error) {
	var deps []*Dependency
	var errs []error

	// Try to resolve workspace paths
	paths, err := de.parser.ResolveWorkspacePath(rs, from.RelativePath, locals)
	if err != nil {
		// Fall back to pattern-based matching
		deps, errs = de.matchByRemoteStateName(from, rs)
		return deps, append(errs, fmt.Errorf("could not resolve workspace path for %s.%s: %w",
			from.ID(), rs.Name, err))
	}

	// Match paths to modules
	for _, path := range paths {
		target := de.matchPathToModule(path, from)
		if target != nil {
			deps = append(deps, &Dependency{
				From:            from,
				To:              target,
				Type:            "remote_state",
				RemoteStateName: rs.Name,
			})
		} else {
			errs = append(errs, fmt.Errorf("could not find module for path %s (from %s.%s)",
				path, from.ID(), rs.Name))
		}
	}

	return deps, errs
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

// matchByRemoteStateName attempts to match by remote state name conventions
func (de *DependencyExtractor) matchByRemoteStateName(from *discovery.Module, rs *RemoteStateRef) ([]*Dependency, []error) {
	var deps []*Dependency
	var errs []error

	// Common naming conventions:
	// - data.terraform_remote_state.vpc -> look for vpc module
	// - data.terraform_remote_state.eks_cluster -> look for eks-cluster or eks_cluster module
	// - data.terraform_remote_state.ec2_rabbitmq -> look for ec2/rabbitmq submodule

	// Normalize the remote state name
	possibleNames := []string{
		rs.Name,
		strings.ReplaceAll(rs.Name, "_", "-"),
		strings.ReplaceAll(rs.Name, "-", "_"),
	}

	// Search in same service/environment/region first (base modules)
	for _, name := range possibleNames {
		sameContextID := fmt.Sprintf("%s/%s/%s/%s",
			from.Service, from.Environment, from.Region, name)
		if m := de.index.ByID(sameContextID); m != nil {
			deps = append(deps, &Dependency{
				From:            from,
				To:              m,
				Type:            "remote_state",
				RemoteStateName: rs.Name,
			})
			return deps, errs
		}
	}

	// Try to match submodule pattern (e.g., ec2_rabbitmq -> ec2/rabbitmq)
	for _, name := range possibleNames {
		// Try splitting by underscore to find module/submodule pattern
		parts := strings.SplitN(name, "_", 2)
		if len(parts) == 2 {
			submoduleID := fmt.Sprintf("%s/%s/%s/%s/%s",
				from.Service, from.Environment, from.Region, parts[0], parts[1])
			if m := de.index.ByID(submoduleID); m != nil {
				deps = append(deps, &Dependency{
					From:            from,
					To:              m,
					Type:            "remote_state",
					RemoteStateName: rs.Name,
				})
				return deps, errs
			}
		}

		// Also try with hyphen
		parts = strings.SplitN(name, "-", 2)
		if len(parts) == 2 {
			submoduleID := fmt.Sprintf("%s/%s/%s/%s/%s",
				from.Service, from.Environment, from.Region, parts[0], parts[1])
			if m := de.index.ByID(submoduleID); m != nil {
				deps = append(deps, &Dependency{
					From:            from,
					To:              m,
					Type:            "remote_state",
					RemoteStateName: rs.Name,
				})
				return deps, errs
			}
		}
	}

	// If we're in a submodule, check sibling submodules first
	if from.IsSubmodule() {
		for _, name := range possibleNames {
			siblingID := fmt.Sprintf("%s/%s/%s/%s/%s",
				from.Service, from.Environment, from.Region, from.Module, name)
			if m := de.index.ByID(siblingID); m != nil {
				deps = append(deps, &Dependency{
					From:            from,
					To:              m,
					Type:            "remote_state",
					RemoteStateName: rs.Name,
				})
				return deps, errs
			}
		}

		// Also check parent module
		parentID := fmt.Sprintf("%s/%s/%s/%s",
			from.Service, from.Environment, from.Region, from.Module)
		if m := de.index.ByID(parentID); m != nil {
			// Check if the remote state name matches parent module name
			for _, name := range possibleNames {
				if name == from.Module {
					deps = append(deps, &Dependency{
						From:            from,
						To:              m,
						Type:            "remote_state",
						RemoteStateName: rs.Name,
					})
					return deps, errs
				}
			}
		}
	}

	// Search across all modules by name
	for _, name := range possibleNames {
		modules := de.index.Filter(func(m *discovery.Module) bool {
			return m.Name() == name && m.ID() != from.ID()
		})

		if len(modules) == 1 {
			deps = append(deps, &Dependency{
				From:            from,
				To:              modules[0],
				Type:            "remote_state",
				RemoteStateName: rs.Name,
			})
			return deps, errs
		} else if len(modules) > 1 {
			// Ambiguous - prefer same environment
			for _, m := range modules {
				if m.Environment == from.Environment {
					deps = append(deps, &Dependency{
						From:            from,
						To:              m,
						Type:            "remote_state",
						RemoteStateName: rs.Name,
					})
					return deps, errs
				}
			}
		}
	}

	errs = append(errs, fmt.Errorf("could not match remote state %s to any module", rs.Name))
	return deps, errs
}

// ExtractAllDependencies extracts dependencies for all modules in the index
func (de *DependencyExtractor) ExtractAllDependencies() (map[string]*ModuleDependencies, []error) {
	results := make(map[string]*ModuleDependencies)
	var allErrors []error

	for _, module := range de.index.All() {
		deps, err := de.ExtractDependencies(module)
		if err != nil {
			allErrors = append(allErrors, err)
			continue
		}

		results[module.ID()] = deps
		allErrors = append(allErrors, deps.Errors...)
	}

	return results, allErrors
}

// PathPatternMatcher helps match state file paths with variables
type PathPatternMatcher struct {
	// Pattern with placeholders like ${local.service}/${local.environment}/${local.region}/${module}/terraform.tfstate
	Pattern string
	// Compiled regex
	regex *regexp.Regexp
	// Group names
	groups []string
}

// NewPathPatternMatcher creates a matcher from a pattern
func NewPathPatternMatcher(pattern string) (*PathPatternMatcher, error) {
	// Convert pattern to regex
	// ${local.service} -> (?P<service>[^/]+)
	// ${local.environment} -> (?P<environment>[^/]+)
	// etc.

	regexPattern := regexp.QuoteMeta(pattern)

	placeholderRe := regexp.MustCompile(`\\\$\\\{local\.(\w+)\\\}`)
	var groups []string

	regexPattern = placeholderRe.ReplaceAllStringFunc(regexPattern, func(match string) string {
		submatches := placeholderRe.FindStringSubmatch(match)
		if len(submatches) >= 2 {
			groupName := submatches[1]
			groups = append(groups, groupName)
			return fmt.Sprintf("(?P<%s>[^/]+)", groupName)
		}
		return match
	})

	// Also handle each.key and each.value
	eachRe := regexp.MustCompile(`\\\$\\\{each\.(key|value)\\\}`)
	regexPattern = eachRe.ReplaceAllStringFunc(regexPattern, func(match string) string {
		groups = append(groups, "each")
		return "(?P<each>[^/]+)"
	})

	compiled, err := regexp.Compile("^" + regexPattern + "$")
	if err != nil {
		return nil, fmt.Errorf("failed to compile pattern regex: %w", err)
	}

	return &PathPatternMatcher{
		Pattern: pattern,
		regex:   compiled,
		groups:  groups,
	}, nil
}

// Match attempts to match a path and extract components
func (m *PathPatternMatcher) Match(path string) (map[string]string, bool) {
	matches := m.regex.FindStringSubmatch(path)
	if matches == nil {
		return nil, false
	}

	result := make(map[string]string)
	for i, name := range m.regex.SubexpNames() {
		if i > 0 && name != "" && i < len(matches) {
			result[name] = matches[i]
		}
	}

	return result, true
}

// ToModuleID converts matched components to a module ID
func (m *PathPatternMatcher) ToModuleID(components map[string]string) string {
	service := components["service"]
	env := components["environment"]
	region := components["region"]
	module := components["module"]

	if service != "" && env != "" && region != "" && module != "" {
		return fmt.Sprintf("%s/%s/%s/%s", service, env, region, module)
	}

	return ""
}
