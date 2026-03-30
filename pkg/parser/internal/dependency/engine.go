package dependency

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/edelwud/terraci/pkg/discovery"
	terrierrors "github.com/edelwud/terraci/pkg/errors"
	parserdeps "github.com/edelwud/terraci/pkg/parser/internal/deps"
	"github.com/edelwud/terraci/pkg/parser/internal/model"
)

const maxConcurrentExtractions = 20

type Engine struct {
	parser       ModuleParser
	index        *discovery.ModuleIndex
	cache        *parsedModuleCache
	backendIndex *backendModuleIndex
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
	return newDependencySession(ctx, e, module).Run()
}

func (e *Engine) ExtractAllDependencies(ctx context.Context) (map[string]*ModuleDependencies, []error) {
	e.backendIndex.Build(ctx, e.index, e.cache)

	builder := newDependencyCollectionBuilder()
	var collector resultCollector

	collector.run(e.index.All(), func(module *discovery.Module) (*ModuleDependencies, error) {
		return e.ExtractDependencies(ctx, module)
	}, func(module *discovery.Module, deps *ModuleDependencies, err error) {
		builder.Add(module, deps, err)
	})

	return builder.Build()
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

type parsedModuleCache struct {
	parser ModuleParser
	items  map[string]*model.ParsedModule
	mu     sync.RWMutex
}

func newParsedModuleCache(parser ModuleParser) *parsedModuleCache {
	return &parsedModuleCache{
		parser: parser,
		items:  make(map[string]*model.ParsedModule),
	}
}

func (c *parsedModuleCache) Get(ctx context.Context, module *discovery.Module) (*model.ParsedModule, error) {
	moduleID := module.ID()

	c.mu.RLock()
	parsed, ok := c.items[moduleID]
	c.mu.RUnlock()
	if ok {
		return parsed, nil
	}

	parsed, err := c.parser.ParseModule(ctx, module.Path)
	if err != nil {
		return nil, &terrierrors.ParseError{Module: module.ID(), Err: err}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if cached, ok := c.items[moduleID]; ok {
		return cached, nil
	}

	c.items[moduleID] = parsed
	return parsed, nil
}

type backendModuleIndex struct {
	items map[string]*discovery.Module
}

func newBackendModuleIndex() *backendModuleIndex {
	return &backendModuleIndex{
		items: make(map[string]*discovery.Module),
	}
}

func (i *backendModuleIndex) Build(ctx context.Context, index *discovery.ModuleIndex, cache *parsedModuleCache) {
	var mu sync.Mutex
	var group errgroup.Group
	group.SetLimit(maxConcurrentExtractions)

	for _, module := range index.All() {
		group.Go(func() error {
			parsed, err := cache.Get(ctx, module)

			mu.Lock()
			defer mu.Unlock()

			if err != nil || parsed.Backend == nil {
				return nil
			}

			key := BackendIndexKey(parsed.Backend, module.RelativePath)
			if key == "" {
				return nil
			}

			i.items[key] = module
			return nil
		})
	}

	_ = group.Wait() //nolint:errcheck
}

func (i *backendModuleIndex) Match(backendType, bucket, statePath string) *discovery.Module {
	return parserdeps.MatchByBackend(i.items, backendType, bucket, statePath)
}

type resultCollector struct{}

func (c resultCollector) run(
	modules []*discovery.Module,
	extract func(*discovery.Module) (*ModuleDependencies, error),
	collect func(*discovery.Module, *ModuleDependencies, error),
) {
	var mu sync.Mutex
	var group errgroup.Group
	group.SetLimit(maxConcurrentExtractions)

	for _, module := range modules {
		group.Go(func() error {
			deps, err := extract(module)

			mu.Lock()
			defer mu.Unlock()
			collect(module, deps, err)
			return nil
		})
	}

	_ = group.Wait() //nolint:errcheck
}
