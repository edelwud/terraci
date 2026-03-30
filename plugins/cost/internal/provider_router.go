package costengine

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
)

// ProviderRouter resolves the owning cloud provider for a Terraform resource type.
type ProviderRouter interface {
	ResolveProvider(resourceType handler.ResourceType) (string, bool)
}

// ResourceProviderRouter is the default resource-type based provider router.
type ResourceProviderRouter struct {
	providers map[handler.ResourceType]string
}

// NewResourceProviderRouter creates an empty provider router.
func NewResourceProviderRouter() *ResourceProviderRouter {
	return &ResourceProviderRouter{
		providers: make(map[handler.ResourceType]string),
	}
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
