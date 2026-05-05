package dependency

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/internal/terraform/eval"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/parser/internal/exprfast"
	"github.com/edelwud/terraci/pkg/parser/model"
)

// DependencyTypeRemoteState marks a dependency derived from a
// terraform_remote_state data source. Reused by tests so the literal stays
// in one place.
const DependencyTypeRemoteState = "remote_state"

type dependencySession struct {
	ctx       context.Context
	deps      sessionDependencies
	module    *discovery.Module
	builder   *dependencyResultBuilder
	parsed    *model.ParsedModule
	locals    map[string]cty.Value
	variables map[string]cty.Value
}

type parsedModuleStore interface {
	Get(context.Context, *discovery.Module) (*model.ParsedModule, error)
}

type workspaceResolver interface {
	ResolveWorkspacePath(ref *model.RemoteStateRef, modulePath string, locals, variables map[string]cty.Value) ([]string, error)
}

type targetMatcher interface {
	MatchPathToModule(statePath string, from *discovery.Module) *discovery.Module
	MatchBackend(ctx context.Context, backendType, bucket, statePath string) *discovery.Module
}

type sessionDependencies struct {
	resolver      workspaceResolver
	parsedModules parsedModuleStore
	targets       targetMatcher
}

func newSessionDependencies(engine *Engine) sessionDependencies {
	return sessionDependencies{
		resolver:      engine.parser,
		parsedModules: engine.cache,
		targets:       engine,
	}
}

func newDependencySession(ctx context.Context, engine *Engine, module *discovery.Module) *dependencySession {
	return &dependencySession{
		ctx:     ctx,
		deps:    newSessionDependencies(engine),
		module:  module,
		builder: newDependencyResultBuilder(module),
	}
}

func (s *dependencySession) Run() (*ModuleDependencies, error) {
	parsed, err := s.deps.parsedModules.Get(s.ctx, s.module)
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

func (s *dependencySession) resolveRemoteStateDependency(remoteState *model.RemoteStateRef) *remoteStateResolution {
	resolution := newRemoteStateResolution()
	targetResolver := newRemoteStateTargetResolver(s.ctx, s.deps.targets, s.module, s.locals, s.variables)

	paths, err := s.deps.resolver.ResolveWorkspacePath(remoteState, s.module.RelativePath, s.locals, s.variables)
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
			Type:            DependencyTypeRemoteState,
			RemoteStateName: remoteState.Name,
		})
	}

	return resolution
}

type remoteStateTargetResolver struct {
	ctx       context.Context
	targets   targetMatcher
	module    *discovery.Module
	locals    map[string]cty.Value
	variables map[string]cty.Value
}

func newRemoteStateTargetResolver(
	ctx context.Context,
	targets targetMatcher,
	module *discovery.Module,
	locals, variables map[string]cty.Value,
) *remoteStateTargetResolver {
	return &remoteStateTargetResolver{
		ctx:       ctx,
		targets:   targets,
		module:    module,
		locals:    locals,
		variables: variables,
	}
}

func (r *remoteStateTargetResolver) Resolve(remoteState *model.RemoteStateRef, statePath string) *discovery.Module {
	target := r.targets.MatchPathToModule(statePath, r.module)
	if target != nil {
		return target
	}

	return r.matchByBackend(remoteState, statePath)
}

func (r *remoteStateTargetResolver) matchByBackend(remoteState *model.RemoteStateRef, statePath string) *discovery.Module {
	if remoteState.Backend == "" {
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

	return r.targets.MatchBackend(r.ctx, remoteState.Backend, bucket, statePath)
}

func evalStringExpr(expr hcl.Expression, ctx *hcl.EvalContext) (string, bool) {
	return exprfast.EvalString(expr, ctx)
}
