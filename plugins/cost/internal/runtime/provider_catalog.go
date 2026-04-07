package runtime

import (
	"maps"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

// ResourceProviderRouter is the default resource-type based provider router.
type ResourceProviderRouter struct {
	providers map[handler.ResourceType]string
}

// NewResourceProviderRouter creates an empty provider router.
func NewResourceProviderRouter() *ResourceProviderRouter {
	return &ResourceProviderRouter{providers: make(map[handler.ResourceType]string)}
}

// Register records the owning provider for a resource type.
func (r *ResourceProviderRouter) Register(providerID string, resourceType handler.ResourceType) {
	r.providers[resourceType] = providerID
}

// ResolveProvider returns the provider id for a resource type.
func (r *ResourceProviderRouter) ResolveProvider(resourceType handler.ResourceType) (string, bool) {
	providerID, ok := r.providers[resourceType]
	return providerID, ok
}

func newDefaultProviderRouter(providers []cloud.Provider) *ResourceProviderRouter {
	router := NewResourceProviderRouter()
	for _, cp := range providers {
		def := cp.Definition()
		for _, resource := range def.Resources {
			router.Register(def.Manifest.ID, resource.Type)
		}
	}
	return router
}

// ProviderCatalog resolves provider ownership, handlers, and provider metadata.
type ProviderCatalog struct {
	registry *handler.Registry
	router   *ResourceProviderRouter
	metadata map[string]model.ProviderMetadata
}

// NewProviderCatalog creates a provider catalog from explicit router, registry, and metadata.
func NewProviderCatalog(router *ResourceProviderRouter, registry *handler.Registry, metadata map[string]model.ProviderMetadata) *ProviderCatalog {
	copiedMetadata := make(map[string]model.ProviderMetadata, len(metadata))
	maps.Copy(copiedMetadata, metadata)

	return &ProviderCatalog{
		registry: registry,
		router:   router,
		metadata: copiedMetadata,
	}
}

// NewProviderCatalogFromProviders creates a provider catalog directly from provider definitions.
func NewProviderCatalogFromProviders(providers []cloud.Provider, registry *handler.Registry) *ProviderCatalog {
	metadata := make(map[string]model.ProviderMetadata, len(providers))
	for _, cp := range providers {
		manifest := cp.Definition().Manifest
		if manifest.ID == "" {
			continue
		}
		metadata[manifest.ID] = model.ProviderMetadata{
			DisplayName: manifest.DisplayName,
			PriceSource: manifest.PriceSource,
		}
	}

	return NewProviderCatalog(newDefaultProviderRouter(providers), registry, metadata)
}

// ResolveProvider returns the owning provider for a resource type.
func (c *ProviderCatalog) ResolveProvider(resourceType handler.ResourceType) (string, bool) {
	if c.router == nil {
		return "", false
	}
	return c.router.ResolveProvider(resourceType)
}

// ResolveHandler returns a provider-scoped resource handler.
func (c *ProviderCatalog) ResolveHandler(providerID string, resourceType handler.ResourceType) (handler.ResourceHandler, bool) {
	if c.registry == nil {
		return nil, false
	}
	return c.registry.ResolveHandler(providerID, resourceType)
}

// ProviderMetadata returns provider-specific estimation metadata keyed by provider id.
func (c *ProviderCatalog) ProviderMetadata() map[string]model.ProviderMetadata {
	if len(c.metadata) == 0 {
		return nil
	}

	meta := make(map[string]model.ProviderMetadata, len(c.metadata))
	maps.Copy(meta, c.metadata)
	return meta
}
