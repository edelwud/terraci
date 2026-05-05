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

// ExtractDependencies resolves dependencies for a single module.
//
// Backend-index construction (which parses every other module in parallel
// to map state-keys to backends) is deferred until MatchBackend is actually
// called from the session. Single-module consumers whose state references
// always disambiguate by path alone don't pay the O(N) cost — they only
// see the index built lazily on first ambiguous lookup.
func (e *Engine) ExtractDependencies(ctx context.Context, module *discovery.Module) (*ModuleDependencies, error) {
	return newDependencySession(ctx, e, module).Run()
}

// ExtractAllDependencies resolves dependencies for every module. Builds the
// backend-index up front because batch traversal will hit it for nearly
// every module anyway — eager construction lets the index parsing run
// concurrently with the first session's setup.
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

// MatchBackend triggers lazy backend-index construction on first use.
// Single-module callers whose remote_state pointers all disambiguate by
// path alone never invoke this — and therefore never pay the O(N) parse cost.
//
// ctx propagates cancellation into the parallel module-parse pass that the
// index uses on first build; subsequent calls reuse the cached index.
func (e *Engine) MatchBackend(ctx context.Context, backendType, bucket, statePath string) *discovery.Module {
	if e.backendIndex == nil {
		return nil
	}

	e.prepareBackendIndex(ctx)

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
