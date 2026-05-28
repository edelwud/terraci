package awskit

import (
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// Runtime exposes provider-owned metadata and helpers to AWS resource specs and definitions.
type Runtime struct {
	Manifest pricing.ProviderManifest
}

// RuntimeDeps stores an optional AWS provider runtime for a resource spec.
type RuntimeDeps struct {
	Runtime *Runtime
}

// NewRuntimeDeps constructs runtime dependencies for AWS resource specs.
func NewRuntimeDeps(runtime *Runtime) RuntimeDeps {
	return RuntimeDeps{Runtime: runtime}
}

// RuntimeOrDefault returns the injected runtime or the default AWS runtime.
func (d RuntimeDeps) RuntimeOrDefault() *Runtime {
	if d.Runtime != nil {
		return d.Runtime
	}
	return DefaultRuntime
}

// NewRuntime constructs a provider runtime from the manifest owned by this provider.
func NewRuntime(manifest pricing.ProviderManifest) *Runtime {
	return &Runtime{Manifest: manifest}
}

// MustService resolves a typed catalog key or panics if the service is not registered.
func (r *Runtime) MustService(key ServiceKey) pricing.ServiceID {
	return r.Manifest.MustService(string(key))
}

// ResolveRegionName returns the AWS pricing API location for a region code.
func (r *Runtime) ResolveRegionName(region string) string {
	return r.Manifest.Regions.ResolveLocationName(region)
}

// ResolveUsagePrefix returns the pricing API usage prefix for a region code.
func (r *Runtime) ResolveUsagePrefix(region string) string {
	return r.Manifest.Regions.ResolveUsagePrefix(region)
}
