package inmemcache

import (
	"context"
	"slices"
	"sync"
	"time"
)

type cache struct {
	mu         sync.Mutex
	namespaces map[string]map[string]entry
}

type entry struct {
	value     []byte
	expiresAt time.Time
}

func newCache() *cache {
	return &cache{
		namespaces: make(map[string]map[string]entry),
	}
}

func (c *cache) Get(_ context.Context, namespace, key string) (value []byte, found bool, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	items, ok := c.namespaces[namespace]
	if !ok {
		return nil, false, nil
	}

	cached, ok := items[key]
	if !ok {
		return nil, false, nil
	}

	if cached.expired() {
		delete(items, key)
		if len(items) == 0 {
			delete(c.namespaces, namespace)
		}
		return nil, false, nil
	}

	return slices.Clone(cached.value), true, nil
}

func (c *cache) Set(_ context.Context, namespace, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	items, ok := c.namespaces[namespace]
	if !ok {
		items = make(map[string]entry)
		c.namespaces[namespace] = items
	}

	items[key] = entry{
		value:     slices.Clone(value),
		expiresAt: expiresAt(ttl),
	}

	return nil
}

func (c *cache) Delete(_ context.Context, namespace, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	items, ok := c.namespaces[namespace]
	if !ok {
		return nil
	}

	delete(items, key)
	if len(items) == 0 {
		delete(c.namespaces, namespace)
	}

	return nil
}

func (c *cache) DeleteNamespace(_ context.Context, namespace string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.namespaces, namespace)
	return nil
}

func expiresAt(ttl time.Duration) time.Time {
	if ttl <= 0 {
		return time.Time{}
	}
	return time.Now().Add(ttl)
}

func (e entry) expired() bool {
	return !e.expiresAt.IsZero() && time.Now().After(e.expiresAt)
}
