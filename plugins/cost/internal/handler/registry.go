// Package handler contains legacy compatibility surfaces that remain useful in tests
// while the canonical runtime contract lives in internal/resourcedef.
package handler

// Registry maps terraform resource types to legacy handler adapters.
// It is retained for compatibility-oriented tests while production estimation
// now resolves canonical resource definitions directly.
type Registry struct {
	handlers map[string]map[ResourceType]ResourceHandler
}

// NewRegistry creates a new empty resource registry.
func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]map[ResourceType]ResourceHandler),
	}
}

// Register adds a handler for a resource type.
func (r *Registry) Register(providerID string, resourceType ResourceType, handler ResourceHandler) {
	if r.handlers[providerID] == nil {
		r.handlers[providerID] = make(map[ResourceType]ResourceHandler)
	}
	r.handlers[providerID][resourceType] = handler
}

// ResolveHandler returns a handler scoped to a provider and resource type.
func (r *Registry) ResolveHandler(providerID string, resourceType ResourceType) (ResourceHandler, bool) {
	providerHandlers, ok := r.handlers[providerID]
	if !ok {
		return nil, false
	}
	h, ok := providerHandlers[resourceType]
	return h, ok
}

// IsSupported checks if a resource type is supported.
func (r *Registry) IsSupported(resourceType ResourceType) bool {
	for _, providerHandlers := range r.handlers {
		if _, ok := providerHandlers[resourceType]; ok {
			return true
		}
	}
	return false
}

// SupportedTypes returns all supported resource types.
func (r *Registry) SupportedTypes() []ResourceType {
	typeSet := make(map[ResourceType]bool)
	for _, providerHandlers := range r.handlers {
		for t := range providerHandlers {
			typeSet[t] = true
		}
	}

	types := make([]ResourceType, 0, len(typeSet))
	for t := range typeSet {
		types = append(types, t)
	}
	return types
}
