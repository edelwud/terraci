package moduleparse

import (
	"context"

	"github.com/hashicorp/hcl/v2"

	"github.com/edelwud/terraci/pkg/parser/internal/evalctx"
	"github.com/edelwud/terraci/pkg/parser/internal/extract"
	"github.com/edelwud/terraci/pkg/parser/internal/source"
)

type Result struct {
	Files          map[string]*hcl.File
	Diagnostics    hcl.Diagnostics
	TopLevelBlocks map[string][]*hcl.Block
}

func Run(ctx context.Context, modulePath string, segments []string, sink extract.Sink) (*Result, error) {
	index, err := source.NewLoader().Load(ctx, modulePath)
	if err != nil {
		return nil, err
	}

	extract.RunDefault(&extract.Context{
		Index:       index,
		EvalBuilder: evalctx.NewBuilder(segments),
		Sink:        sink,
	})

	return &Result{
		Files:          index.Files(),
		Diagnostics:    index.Diagnostics(),
		TopLevelBlocks: index.TopLevelBlockIndex(),
	}, nil
}
