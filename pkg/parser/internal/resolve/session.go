package resolve

import (
	"errors"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/pkg/log"
)

type resolveSession struct {
	resolver   Resolver
	ref        *Ref
	modulePath string
	locals     map[string]cty.Value
	variables  map[string]cty.Value
}

func newResolveSession(
	resolver Resolver,
	ref *Ref,
	modulePath string,
	locals, variables map[string]cty.Value,
) *resolveSession {
	return &resolveSession{
		resolver:   resolver,
		ref:        ref,
		modulePath: modulePath,
		locals:     locals,
		variables:  variables,
	}
}

func (s *resolveSession) Run() ([]string, error) {
	if s.ref == nil {
		return nil, errors.New("remote state ref is nil")
	}

	log.WithField("module", s.modulePath).WithField("remote_state", s.ref.Name).Debug("resolving workspace path")

	evalCtx := s.resolver.evalBuilder.Build(s.modulePath, s.locals, s.variables)
	pathExpr := s.pathExpression()
	if pathExpr == nil {
		return nil, errors.New("no key or prefix found in remote state config")
	}

	if s.ref.ForEach != nil {
		return s.resolver.resolveForEach(s.ref.ForEach, pathExpr, evalCtx)
	}

	return s.resolver.resolveSimple(pathExpr, evalCtx)
}

func (s *resolveSession) pathExpression() hcl.Expression {
	return findPathExpression(s.ref)
}
