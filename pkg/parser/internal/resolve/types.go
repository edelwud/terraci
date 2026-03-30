package resolve

import (
	"github.com/edelwud/terraci/pkg/parser/internal/evalctx"
	"github.com/edelwud/terraci/pkg/parser/model"
)

type Resolver struct {
	evalBuilder evalctx.Builder
}

func NewResolver(evalBuilder evalctx.Builder) Resolver {
	return Resolver{evalBuilder: evalBuilder}
}

type Ref = model.RemoteStateRef
