package dependency

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/edelwud/terraci/pkg/discovery"
	parserdeps "github.com/edelwud/terraci/pkg/parser/internal/deps"
)

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
