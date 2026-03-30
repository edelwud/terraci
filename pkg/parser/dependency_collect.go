package parser

import (
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/edelwud/terraci/pkg/discovery"
)

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
