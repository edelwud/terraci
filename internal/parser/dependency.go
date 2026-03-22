package parser

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/zclconf/go-cty/cty"
	"golang.org/x/sync/errgroup"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/terraform/eval"
	terrierrors "github.com/edelwud/terraci/pkg/errors"
)

// DependencyExtractor extracts module dependencies from parsed Terraform files.
type DependencyExtractor struct {
	parser       ModuleParser
	index        *discovery.ModuleIndex
	parsedCache  map[string]*ParsedModule
	backendIndex map[string]*discovery.Module
}

// NewDependencyExtractor creates a new dependency extractor.
func NewDependencyExtractor(parser ModuleParser, index *discovery.ModuleIndex) *DependencyExtractor {
	return &DependencyExtractor{
		parser:      parser,
		index:       index,
		parsedCache: make(map[string]*ParsedModule),
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
	result := &ModuleDependencies{
		Module:              module,
		Dependencies:        make([]*Dependency, 0),
		LibraryDependencies: make([]*LibraryDependency, 0),
		DependsOn:           make([]string, 0),
		Errors:              make([]error, 0),
	}

	parsed, err := de.parseModule(ctx, module)
	if err != nil {
		return nil, &terrierrors.ParseError{Module: module.ID(), Err: err}
	}

	for _, rs := range parsed.RemoteStates {
		deps, errs := de.resolveRemoteStateDependency(module, rs, parsed.Locals, parsed.Variables)
		result.Dependencies = append(result.Dependencies, deps...)
		result.Errors = append(result.Errors, errs...)
	}

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

// resolveRemoteStateDependency resolves a remote state to module dependencies.
func (de *DependencyExtractor) resolveRemoteStateDependency(
	from *discovery.Module, rs *RemoteStateRef,
	locals, variables map[string]cty.Value,
) ([]*Dependency, []error) {
	var deps []*Dependency
	var errs []error

	paths, err := de.parser.ResolveWorkspacePath(rs, from.RelativePath, locals, variables)
	if err != nil {
		errs = append(errs, fmt.Errorf("resolve workspace path for %s.%s: %w", from.ID(), rs.Name, err))
		return deps, errs
	}

	for _, path := range paths {
		if containsDynamicPattern(path) {
			errs = append(errs, fmt.Errorf("unresolved dynamic path %q for %s.%s", path, from.ID(), rs.Name))
			continue
		}

		target := de.matchPathToModule(path, from)
		if target == nil {
			target = de.matchByBackend(rs, path, locals, variables, from)
		}

		if target != nil {
			deps = append(deps, &Dependency{
				From:            from,
				To:              target,
				Type:            "remote_state",
				RemoteStateName: rs.Name,
			})
		} else {
			errs = append(errs, fmt.Errorf("no module for path %q (from %s.%s)", path, from.ID(), rs.Name))
		}
	}

	return deps, errs
}

// matchPathToModule matches a state file path to a module using multiple strategies.
func (de *DependencyExtractor) matchPathToModule(statePath string, from *discovery.Module) *discovery.Module {
	normalized := normalizeStatePath(statePath)
	parts := strings.Split(normalized, "/")

	// Strategy chain: try each approach, return first match
	strategies := []func() *discovery.Module{
		func() *discovery.Module { return de.index.ByID(normalized) },
		func() *discovery.Module {
			return de.index.ByID(strings.ReplaceAll(normalized, "/", string(filepath.Separator)))
		},
		func() *discovery.Module { return de.tryTrailingMatch(parts, 5) },
		func() *discovery.Module { return de.tryTrailingMatch(parts, 4) },
		func() *discovery.Module { return de.tryContextMatch(parts, from) },
	}

	for _, s := range strategies {
		if m := s(); m != nil {
			return m
		}
	}
	return nil
}

// normalizeStatePath strips state file suffixes and env prefixes.
func normalizeStatePath(path string) string {
	path = strings.TrimSuffix(path, "/terraform.tfstate")
	path = strings.TrimSuffix(path, ".tfstate")
	path = strings.TrimPrefix(path, "env:/")
	return path
}

// tryTrailingMatch tries to match the last N parts of the path as a module ID.
func (de *DependencyExtractor) tryTrailingMatch(parts []string, n int) *discovery.Module {
	if len(parts) < n {
		return nil
	}
	return de.index.ByID(strings.Join(parts[len(parts)-n:], "/"))
}

// tryContextMatch tries context-relative matching using the source module's context.
func (de *DependencyExtractor) tryContextMatch(parts []string, from *discovery.Module) *discovery.Module {
	prefix := from.ContextPrefix()

	if len(parts) == 1 {
		if m := de.index.ByID(prefix + "/" + parts[0]); m != nil {
			return m
		}
		// Try as sibling submodule
		if from.IsSubmodule() {
			if m := de.index.ByID(prefix + "/" + from.LeafValue() + "/" + parts[0]); m != nil {
				return m
			}
		}
	}

	if len(parts) == 2 {
		return de.index.ByID(prefix + "/" + parts[0] + "/" + parts[1])
	}

	return nil
}

func containsDynamicPattern(path string) bool {
	return strings.Contains(path, "${lookup(") ||
		strings.Contains(path, "${each.") ||
		strings.Contains(path, "${var.") ||
		strings.Contains(path, "\"}")
}

// parseModule returns a cached ParsedModule or parses it fresh.
func (de *DependencyExtractor) parseModule(ctx context.Context, module *discovery.Module) (*ParsedModule, error) {
	if pm, ok := de.parsedCache[module.ID()]; ok {
		return pm, nil
	}
	pm, err := de.parser.ParseModule(ctx, module.Path)
	if err != nil {
		return nil, err
	}
	de.parsedCache[module.ID()] = pm
	return pm, nil
}

// buildBackendIndex pre-parses all modules and builds a backend config index.
func (de *DependencyExtractor) buildBackendIndex(ctx context.Context) {
	var mu sync.Mutex
	var g errgroup.Group
	g.SetLimit(maxConcurrentExtractions)

	for _, module := range de.index.All() {
		g.Go(func() error {
			pm, err := de.parser.ParseModule(ctx, module.Path)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				return nil
			}
			de.parsedCache[module.ID()] = pm

			if pm.Backend != nil {
				key := backendIndexKey(pm.Backend, module.RelativePath)
				if key != "" {
					if de.backendIndex == nil {
						de.backendIndex = make(map[string]*discovery.Module)
					}
					de.backendIndex[key] = module
				}
			}
			return nil
		})
	}
	_ = g.Wait() //nolint:errcheck
}

// backendIndexKey builds a lookup key from a module's backend config.
func backendIndexKey(bc *BackendConfig, modulePath string) string {
	bucket := bc.Config["bucket"]
	if bucket == "" {
		return ""
	}
	// Use the module's own backend key if available, otherwise derive from module path
	stateKey := bc.Config["key"]
	if stateKey != "" {
		stateKey = normalizeStatePath(stateKey)
	} else {
		stateKey = modulePath
	}
	return bc.Type + ":" + bucket + ":" + stateKey
}

// matchByBackend tries to find a target module using backend config attributes.
func (de *DependencyExtractor) matchByBackend(
	rs *RemoteStateRef, statePath string,
	locals, variables map[string]cty.Value, from *discovery.Module,
) *discovery.Module {
	if de.backendIndex == nil || rs.Backend == "" {
		return nil
	}

	bucketExpr, ok := rs.Config["bucket"]
	if !ok {
		return nil
	}

	evalCtx := eval.NewContext(locals, variables, from.Path)
	bucket, ok := evalStringExpr(bucketExpr, evalCtx)
	if !ok {
		return nil
	}

	normalized := normalizeStatePath(statePath)
	key := rs.Backend + ":" + bucket + ":" + normalized
	return de.backendIndex[key]
}

// maxConcurrentExtractions is the maximum number of concurrent module extractions.
const maxConcurrentExtractions = 20

// ExtractAllDependencies extracts dependencies for all modules in the index.
func (de *DependencyExtractor) ExtractAllDependencies(ctx context.Context) (map[string]*ModuleDependencies, []error) {
	de.buildBackendIndex(ctx)

	results := make(map[string]*ModuleDependencies)
	var allErrors []error
	var mu sync.Mutex

	var g errgroup.Group
	g.SetLimit(maxConcurrentExtractions)

	for _, module := range de.index.All() {
		g.Go(func() error {
			deps, err := de.ExtractDependencies(ctx, module)

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

	_ = g.Wait() //nolint:errcheck
	return results, allErrors
}
