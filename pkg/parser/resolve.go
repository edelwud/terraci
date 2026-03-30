package parser

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/edelwud/terraci/pkg/parser/internal/resolve"
)

// ResolveWorkspacePath resolves workspace paths from a remote state config
// using the module's locals, variables, and path-derived segment values.
func (p *Parser) ResolveWorkspacePath(ref *RemoteStateRef, modulePath string, locals, variables map[string]cty.Value) ([]string, error) {
	return resolve.NewResolver(p.evalContextBuilder().inner).Resolve(resolve.Ref{
		Name:    ref.Name,
		Config:  ref.Config,
		ForEach: ref.ForEach,
	}, modulePath, locals, variables)
}
