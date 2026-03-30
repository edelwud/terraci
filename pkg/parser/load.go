package parser

import (
	"context"

	"github.com/edelwud/terraci/pkg/parser/internal/source"
)

type moduleLoader struct {
	inner *source.Loader
}

func newModuleLoader() *moduleLoader {
	return &moduleLoader{inner: source.NewLoader()}
}

func (l *moduleLoader) Load(ctx context.Context, modulePath string) (*moduleIndex, error) {
	index, err := l.inner.Load(ctx, modulePath)
	if err != nil {
		return nil, err
	}
	return newModuleIndex(index), nil
}
