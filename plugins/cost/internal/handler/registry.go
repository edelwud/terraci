package handler

import (
	"github.com/caarlos0/log"
)

// Registry maps terraform resource types to cost estimation handlers.
// Provider-agnostic: AWS, GCP, Azure handlers all register here.
type Registry struct {
	handlers map[ResourceType]RegisteredHandler
}

// RegisteredHandler keeps the owning provider id alongside the handler.
type RegisteredHandler struct {
	Provider string
	Handler  ResourceHandler
}

// NewRegistry creates a new empty resource registry.
func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[ResourceType]RegisteredHandler),
	}
}

// Register adds a handler for a resource type.
func (r *Registry) Register(providerID string, resourceType ResourceType, handler ResourceHandler) {
	r.handlers[resourceType] = RegisteredHandler{
		Provider: providerID,
		Handler:  handler,
	}
}

// GetHandler returns a handler for a resource type.
func (r *Registry) GetHandler(resourceType ResourceType) (ResourceHandler, bool) {
	h, ok := r.handlers[resourceType]
	return h.Handler, ok
}

// Resolve returns the registered provider id and handler for a resource type.
func (r *Registry) Resolve(resourceType ResourceType) (RegisteredHandler, bool) {
	h, ok := r.handlers[resourceType]
	return h, ok
}

// IsSupported checks if a resource type is supported.
func (r *Registry) IsSupported(resourceType ResourceType) bool {
	_, ok := r.handlers[resourceType]
	return ok
}

// SupportedTypes returns all supported resource types.
func (r *Registry) SupportedTypes() []string {
	types := make([]string, 0, len(r.handlers))
	for t := range r.handlers {
		types = append(types, t.String())
	}
	return types
}

// LogUnsupported logs unsupported resource types at debug level.
func LogUnsupported(resourceType, address string) {
	log.WithField("type", resourceType).
		WithField("address", address).
		Debug("resource type not supported for cost estimation")
}
