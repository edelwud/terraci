package handler

import (
	"github.com/caarlos0/log"

	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// Registry maps terraform resource types to cost estimation handlers.
// Provider-agnostic: AWS, GCP, Azure handlers all register here.
type Registry struct {
	handlers map[string]RegisteredHandler
}

// RegisteredHandler keeps the owning provider id alongside the handler.
type RegisteredHandler struct {
	Provider string
	Handler  ResourceHandler
}

// NewRegistry creates a new empty resource registry.
func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]RegisteredHandler),
	}
}

// Register adds a handler for a resource type.
func (r *Registry) Register(providerID, resourceType string, handler ResourceHandler) {
	r.handlers[resourceType] = RegisteredHandler{
		Provider: providerID,
		Handler:  handler,
	}
}

// GetHandler returns a handler for a resource type.
func (r *Registry) GetHandler(resourceType string) (ResourceHandler, bool) {
	h, ok := r.handlers[resourceType]
	return h.Handler, ok
}

// Resolve returns the registered provider id and handler for a resource type.
func (r *Registry) Resolve(resourceType string) (RegisteredHandler, bool) {
	h, ok := r.handlers[resourceType]
	return h, ok
}

// IsSupported checks if a resource type is supported.
func (r *Registry) IsSupported(resourceType string) bool {
	_, ok := r.handlers[resourceType]
	return ok
}

// SupportedTypes returns all supported resource types.
func (r *Registry) SupportedTypes() []string {
	types := make([]string, 0, len(r.handlers))
	for t := range r.handlers {
		types = append(types, t)
	}
	return types
}

// RequiredServices returns services needed for given resource types.
func (r *Registry) RequiredServices(resourceTypes []string) map[pricing.ServiceID]bool {
	services := make(map[pricing.ServiceID]bool)
	for _, rt := range resourceTypes {
		if h, ok := r.handlers[rt]; ok {
			services[h.Handler.ServiceCode()] = true
		}
	}
	return services
}

// LogUnsupported logs unsupported resource types at debug level.
func LogUnsupported(resourceType, address string) {
	log.WithField("type", resourceType).
		WithField("address", address).
		Debug("resource type not supported for cost estimation")
}
