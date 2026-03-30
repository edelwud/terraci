package dependency

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/internal/terraform/eval"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/parser/internal/exprfast"
)

type dependencySession struct {
	ctx       context.Context
	engine    *Engine
	module    *discovery.Module
	builder   *dependencyResultBuilder
	parsed    *ParsedModule
	locals    map[string]cty.Value
	variables map[string]cty.Value
}

func newDependencySession(ctx context.Context, engine *Engine, module *discovery.Module) *dependencySession {
	return &dependencySession{
		ctx:     ctx,
		engine:  engine,
		module:  module,
		builder: newDependencyResultBuilder(module),
	}
}

func (s *dependencySession) Run() (*ModuleDependencies, error) {
	parsed, err := s.engine.cache.Get(s.ctx, s.module)
	if err != nil {
		return nil, err
	}

	s.parsed = parsed
	s.locals = parsed.Locals
	s.variables = parsed.Variables

	s.collectRemoteStateDependencies()
	s.collectLibraryDependencies()

	return s.builder.Build(), nil
}

func (s *dependencySession) collectRemoteStateDependencies() {
	for _, remoteState := range s.parsed.RemoteStates {
		resolution := s.resolveRemoteStateDependency(remoteState)
		s.builder.AddDependencies(resolution.Dependencies()...)
		s.builder.AddErrors(resolution.Errors()...)
	}
}

func (s *dependencySession) collectLibraryDependencies() {
	for _, moduleCall := range s.parsed.ModuleCalls {
		if !moduleCall.IsLocal || moduleCall.ResolvedPath == "" {
			continue
		}

		s.builder.AddLibraryDependency(&LibraryDependency{
			ModuleCall:  moduleCall,
			LibraryPath: moduleCall.ResolvedPath,
		})
	}
}

func (s *dependencySession) resolveRemoteStateDependency(remoteState *RemoteStateRef) *remoteStateResolution {
	resolution := newRemoteStateResolution()
	targetResolver := newRemoteStateTargetResolver(s.engine, s.module, s.locals, s.variables)

	paths, err := s.engine.parser.ResolveWorkspacePath(remoteState, s.module.RelativePath, s.locals, s.variables)
	if err != nil {
		resolution.AddError(fmt.Errorf("resolve workspace path for %s.%s: %w", s.module.ID(), remoteState.Name, err))
		return resolution
	}

	for _, path := range paths {
		if ContainsDynamicPattern(path) {
			resolution.AddError(fmt.Errorf("unresolved dynamic path %q for %s.%s", path, s.module.ID(), remoteState.Name))
			continue
		}

		target := targetResolver.Resolve(remoteState, path)
		if target == nil {
			resolution.AddError(fmt.Errorf("no module for path %q (from %s.%s)", path, s.module.ID(), remoteState.Name))
			continue
		}

		resolution.AddDependency(&Dependency{
			From:            s.module,
			To:              target,
			Type:            "remote_state",
			RemoteStateName: remoteState.Name,
		})
	}

	return resolution
}

type remoteStateTargetResolver struct {
	engine    *Engine
	module    *discovery.Module
	locals    map[string]cty.Value
	variables map[string]cty.Value
}

func newRemoteStateTargetResolver(
	engine *Engine,
	module *discovery.Module,
	locals, variables map[string]cty.Value,
) *remoteStateTargetResolver {
	return &remoteStateTargetResolver{
		engine:    engine,
		module:    module,
		locals:    locals,
		variables: variables,
	}
}

func (r *remoteStateTargetResolver) Resolve(remoteState *RemoteStateRef, statePath string) *discovery.Module {
	target := r.engine.MatchPathToModule(statePath, r.module)
	if target != nil {
		return target
	}

	return r.matchByBackend(remoteState, statePath)
}

func (r *remoteStateTargetResolver) matchByBackend(remoteState *RemoteStateRef, statePath string) *discovery.Module {
	if r.engine.backendIndex == nil || remoteState.Backend == "" {
		return nil
	}

	bucketExpr, ok := remoteState.Config["bucket"]
	if !ok {
		return nil
	}

	evalCtx := eval.NewContext(r.locals, r.variables, r.module.Path)
	bucket, ok := evalStringExpr(bucketExpr, evalCtx)
	if !ok {
		return nil
	}

	return r.engine.backendIndex.Match(remoteState.Backend, bucket, statePath)
}

func evalStringExpr(expr hcl.Expression, ctx *hcl.EvalContext) (string, bool) {
	return exprfast.EvalString(expr, ctx)
}
