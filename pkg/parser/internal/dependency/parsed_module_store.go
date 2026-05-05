package dependency

import (
	"context"
	"fmt"
	"sync"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/parser/model"
)

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
		return nil, fmt.Errorf("parse %s: %w", module.ID(), err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if cached, ok := c.items[moduleID]; ok {
		return cached, nil
	}

	c.items[moduleID] = parsed
	return parsed, nil
}
