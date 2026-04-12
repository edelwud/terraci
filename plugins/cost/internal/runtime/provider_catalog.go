package runtime

import (
	"maps"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
)

// ResourceProviderRouter is the default resource-type based provider router.
type ResourceProviderRouter struct {
	providers map[resourcedef.ResourceType]string
}

// NewResourceProviderRouter creates an empty provider router.
func NewResourceProviderRouter() *ResourceProviderRouter {
	return &ResourceProviderRouter{providers: make(map[resourcedef.ResourceType]string)}
}

// Register records the owning provider for a resource type.
func (r *ResourceProviderRouter) Register(providerID string, resourceType resourcedef.ResourceType) {
	r.providers[resourceType] = providerID
}

// ResolveProvider returns the provider id for a resource type.
func (r *ResourceProviderRouter) ResolveProvider(resourceType resourcedef.ResourceType) (string, bool) {
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

// ProviderCatalog resolves provider ownership, resource definitions, and provider metadata.
type ProviderCatalog struct {
	defs     map[string]map[resourcedef.ResourceType]resourcedef.Definition
	router   *ResourceProviderRouter
	metadata map[string]model.ProviderMetadata
}

// NewProviderCatalog creates a provider catalog from explicit router, resource definitions, and metadata.
func NewProviderCatalog(router *ResourceProviderRouter, defs map[string]map[resourcedef.ResourceType]resourcedef.Definition, metadata map[string]model.ProviderMetadata) *ProviderCatalog {
	copiedMetadata := make(map[string]model.ProviderMetadata, len(metadata))
	maps.Copy(copiedMetadata, metadata)

	copiedDefs := make(map[string]map[resourcedef.ResourceType]resourcedef.Definition, len(defs))
	for providerID, providerDefs := range defs {
		copiedDefs[providerID] = maps.Clone(providerDefs)
	}

	return &ProviderCatalog{
		defs:     copiedDefs,
		router:   router,
		metadata: copiedMetadata,
	}
}

// NewProviderCatalogFromProviders creates a provider catalog directly from provider definitions.
func NewProviderCatalogFromProviders(providers []cloud.Provider) *ProviderCatalog {
	defs := make(map[string]map[resourcedef.ResourceType]resourcedef.Definition, len(providers))
	metadata := make(map[string]model.ProviderMetadata, len(providers))
	for _, cp := range providers {
		definition := cp.Definition()
		manifest := definition.Manifest
		if manifest.ID == "" {
			continue
		}
		providerDefs := make(map[resourcedef.ResourceType]resourcedef.Definition, len(definition.Resources))
		for _, resource := range definition.Resources {
			providerDefs[resource.Type] = resource.Definition
		}
		defs[manifest.ID] = providerDefs
		metadata[manifest.ID] = model.ProviderMetadata{
			DisplayName: manifest.DisplayName,
			PriceSource: manifest.PriceSource,
		}
	}

	return NewProviderCatalog(newDefaultProviderRouter(providers), defs, metadata)
}

// ResolveProvider returns the owning provider for a resource type.
func (c *ProviderCatalog) ResolveProvider(resourceType resourcedef.ResourceType) (string, bool) {
	if c.router == nil {
		return "", false
	}
	return c.router.ResolveProvider(resourceType)
}

// ResolveDefinition returns a provider-scoped canonical resource definition.
func (c *ProviderCatalog) ResolveDefinition(providerID string, resourceType resourcedef.ResourceType) (resourcedef.Definition, bool) {
	if c.defs == nil {
		return resourcedef.Definition{}, false
	}
	providerDefs, ok := c.defs[providerID]
	if !ok {
		return resourcedef.Definition{}, false
	}
	def, ok := providerDefs[resourceType]
	return def, ok
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
