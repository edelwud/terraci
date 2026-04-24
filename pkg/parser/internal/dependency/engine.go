package dependency

import (
	"context"
	"sync"

	"github.com/edelwud/terraci/pkg/discovery"
	parserdeps "github.com/edelwud/terraci/pkg/parser/internal/deps"
	"github.com/edelwud/terraci/pkg/parser/model"
)

const maxConcurrentExtractions = 20

type Engine struct {
	parser       ModuleParser
	index        *discovery.ModuleIndex
	cache        *parsedModuleCache
	backendIndex *backendModuleIndex
	backendOnce  sync.Once
}

func NewEngine(parser ModuleParser, index *discovery.ModuleIndex) *Engine {
	return &Engine{
		parser:       parser,
		index:        index,
		cache:        newParsedModuleCache(parser),
		backendIndex: newBackendModuleIndex(),
	}
}

func (e *Engine) ExtractDependencies(ctx context.Context, module *discovery.Module) (*ModuleDependencies, error) {
	e.prepareBackendIndex(ctx)
	return newDependencySession(ctx, e, module).Run()
}

func (e *Engine) ExtractAllDependencies(ctx context.Context) (map[string]*ModuleDependencies, []error) {
	e.prepareBackendIndex(ctx)

	builder := newDependencyCollectionBuilder()
	var collector resultCollector

	collector.run(e.index.All(), func(module *discovery.Module) (*ModuleDependencies, error) {
		return e.ExtractDependencies(ctx, module)
	}, func(module *discovery.Module, deps *ModuleDependencies, err error) {
		builder.Add(module, deps, err)
	})

	return builder.Build()
}

func (e *Engine) prepareBackendIndex(ctx context.Context) {
	e.backendOnce.Do(func() {
		e.backendIndex.Build(ctx, e.index, e.cache)
	})
}

func (e *Engine) MatchPathToModule(statePath string, from *discovery.Module) *discovery.Module {
	return parserdeps.MatchPathToModule(e.index, statePath, from)
}

func (e *Engine) MatchBackend(backendType, bucket, statePath string) *discovery.Module {
	if e.backendIndex == nil {
		return nil
	}

	return e.backendIndex.Match(backendType, bucket, statePath)
}

func ContainsDynamicPattern(path string) bool {
	return parserdeps.ContainsDynamicPattern(path)
}

func BackendIndexKey(bc *model.BackendConfig, modulePath string) string {
	if bc == nil {
		return ""
	}

	return parserdeps.BackendIndexKey(bc.Type, bc.Config["bucket"], bc.Config["key"], modulePath)
}
