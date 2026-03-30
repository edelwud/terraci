package parser

import (
	"context"
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/internal/terraform/eval"
	"github.com/edelwud/terraci/pkg/discovery"
	terrierrors "github.com/edelwud/terraci/pkg/errors"
	parserdeps "github.com/edelwud/terraci/pkg/parser/internal/deps"
)

type dependencySession struct {
	ctx       context.Context
	extractor *DependencyExtractor
	module    *discovery.Module
	result    *ModuleDependencies
	parsed    *ParsedModule
}

func newDependencySession(ctx context.Context, extractor *DependencyExtractor, module *discovery.Module) *dependencySession {
	return &dependencySession{
		ctx:       ctx,
		extractor: extractor,
		module:    module,
		result: &ModuleDependencies{
			Module:              module,
			Dependencies:        make([]*Dependency, 0),
			LibraryDependencies: make([]*LibraryDependency, 0),
			DependsOn:           make([]string, 0),
			Errors:              make([]error, 0),
		},
	}
}

func (s *dependencySession) Run() (*ModuleDependencies, error) {
	parsed, err := s.extractor.cache.Get(s.ctx, s.module)
	if err != nil {
		return nil, &terrierrors.ParseError{Module: s.module.ID(), Err: err}
	}
	s.parsed = parsed

	s.collectRemoteStateDependencies()
	s.collectLibraryDependencies()
	s.collectDependsOn()

	return s.result, nil
}

func (s *dependencySession) collectRemoteStateDependencies() {
	for _, remoteState := range s.parsed.RemoteStates {
		dependencies, errs := s.resolveRemoteStateDependency(remoteState, s.parsed.Locals, s.parsed.Variables)
		s.result.Dependencies = append(s.result.Dependencies, dependencies...)
		s.result.Errors = append(s.result.Errors, errs...)
	}
}

func (s *dependencySession) collectLibraryDependencies() {
	for _, moduleCall := range s.parsed.ModuleCalls {
		if !moduleCall.IsLocal || moduleCall.ResolvedPath == "" {
			continue
		}
		s.result.LibraryDependencies = append(s.result.LibraryDependencies, &LibraryDependency{
			ModuleCall:  moduleCall,
			LibraryPath: moduleCall.ResolvedPath,
		})
	}
}

func (s *dependencySession) collectDependsOn() {
	seen := make(map[string]bool)
	for _, dependency := range s.result.Dependencies {
		if dependency.To == nil || seen[dependency.To.ID()] {
			continue
		}
		seen[dependency.To.ID()] = true
		s.result.DependsOn = append(s.result.DependsOn, dependency.To.ID())
	}
}

func (s *dependencySession) resolveRemoteStateDependency(
	remoteState *RemoteStateRef,
	locals, variables map[string]cty.Value,
) ([]*Dependency, []error) {
	var dependencies []*Dependency
	var errs []error

	paths, err := s.extractor.parser.ResolveWorkspacePath(remoteState, s.module.RelativePath, locals, variables)
	if err != nil {
		errs = append(errs, fmt.Errorf("resolve workspace path for %s.%s: %w", s.module.ID(), remoteState.Name, err))
		return dependencies, errs
	}

	for _, path := range paths {
		if parserdeps.ContainsDynamicPattern(path) {
			errs = append(errs, fmt.Errorf("unresolved dynamic path %q for %s.%s", path, s.module.ID(), remoteState.Name))
			continue
		}

		target := s.extractor.matchPathToModule(path, s.module)
		if target == nil {
			target = s.matchByBackend(remoteState, path, locals, variables)
		}

		if target == nil {
			errs = append(errs, fmt.Errorf("no module for path %q (from %s.%s)", path, s.module.ID(), remoteState.Name))
			continue
		}

		dependencies = append(dependencies, &Dependency{
			From:            s.module,
			To:              target,
			Type:            "remote_state",
			RemoteStateName: remoteState.Name,
		})
	}

	return dependencies, errs
}

func (s *dependencySession) matchByBackend(
	remoteState *RemoteStateRef,
	statePath string,
	locals, variables map[string]cty.Value,
) *discovery.Module {
	if s.extractor.backendIndex == nil || remoteState.Backend == "" {
		return nil
	}

	bucketExpr, ok := remoteState.Config["bucket"]
	if !ok {
		return nil
	}

	evalCtx := eval.NewContext(locals, variables, s.module.Path)
	bucket, ok := evalStringExpr(bucketExpr, evalCtx)
	if !ok {
		return nil
	}

	return s.extractor.backendIndex.Match(remoteState.Backend, bucket, statePath)
}
