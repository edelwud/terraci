package resolve

import (
	"github.com/zclconf/go-cty/cty"
)

func (r Resolver) Resolve(
	ref *Ref,
	modulePath string,
	locals, variables map[string]cty.Value,
) ([]string, error) {
	return newResolveSession(r, ref, modulePath, locals, variables).Run()
}
