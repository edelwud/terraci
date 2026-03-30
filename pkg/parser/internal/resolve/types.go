package resolve

import (
	"github.com/hashicorp/hcl/v2"

	"github.com/edelwud/terraci/pkg/parser/internal/evalctx"
)

type Ref struct {
	Name    string
	Config  map[string]hcl.Expression
	ForEach hcl.Expression
}

type Resolver struct {
	evalBuilder evalctx.Builder
}

func NewResolver(evalBuilder evalctx.Builder) Resolver {
	return Resolver{evalBuilder: evalBuilder}
}
