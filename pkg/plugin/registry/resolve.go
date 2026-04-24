package registry

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
)

// ciProviderPlugin is the minimum interface set for a CI provider plugin.
// CommentServiceFactory is optional — checked via type assertion in buildResolvedCIProvider.
type ciProviderPlugin interface {
	plugin.Plugin
	plugin.EnvDetector
	plugin.CIInfoProvider
	plugin.PipelineGeneratorFactory
}

func (r *Registry) activeCIProviders() []ciProviderPlugin {
	candidates := ByCapabilityFrom[ciProviderPlugin](r)
	active := make([]ciProviderPlugin, 0, len(candidates))
	for _, c := range candidates {
		if isPluginEnabled(c) {
			active = append(active, c)
		}
	}
	return active
}

// ResolveCIProvider detects the active CI provider.
// Priority: TERRACI_PROVIDER env → env detection → single active provider.
func ResolveCIProvider() (*plugin.ResolvedCIProvider, error) {
	return Default().ResolveCIProvider()
}

// ResolveCIProvider detects the active CI provider in this registry.
// Priority: TERRACI_PROVIDER env → env detection → single active provider.
func (r *Registry) ResolveCIProvider() (*plugin.ResolvedCIProvider, error) {
	candidates := r.activeCIProviders()
	if len(candidates) == 0 {
		return nil, errors.New("no active CI provider plugins registered")
	}

	// Explicit selection wins over auto-detection. This is important for local
	// debugging inside CI-like environments and keeps CLI/env overrides predictable.
	if name := os.Getenv("TERRACI_PROVIDER"); name != "" {
		return findProvider(candidates, name)
	}

	// Check env detection (CI environment variables)
	for _, c := range candidates {
		if c.DetectEnv() {
			return buildResolvedCIProvider(c), nil
		}
	}

	// Single active provider registered
	if len(candidates) == 1 {
		return buildResolvedCIProvider(candidates[0]), nil
	}

	return nil, fmt.Errorf("cannot determine CI provider: multiple plugins registered (%s), set TERRACI_PROVIDER", providerNames(candidates))
}

func buildResolvedCIProvider(p ciProviderPlugin) *plugin.ResolvedCIProvider {
	var comment plugin.CommentServiceFactory
	if cf, ok := p.(plugin.CommentServiceFactory); ok {
		comment = cf
	}
	return plugin.NewResolvedCIProvider(p, p, p, comment)
}

// ResolveChangeDetector returns the active ChangeDetectionProvider.
// Priority: single active detector → error.
func ResolveChangeDetector() (plugin.ChangeDetectionProvider, error) {
	return Default().ResolveChangeDetector()
}

// ResolveChangeDetector returns the active ChangeDetectionProvider in this registry.
// Priority: single active detector → error.
func (r *Registry) ResolveChangeDetector() (plugin.ChangeDetectionProvider, error) {
	detectors := r.activeChangeDetectors()
	if len(detectors) == 0 {
		return nil, errors.New("no change detection plugin registered")
	}
	if len(detectors) == 1 {
		return detectors[0], nil
	}
	return nil, fmt.Errorf("cannot determine change detector: multiple plugins registered (%s)",
		detectorNames(detectors))
}

func (r *Registry) activeChangeDetectors() []plugin.ChangeDetectionProvider {
	candidates := ByCapabilityFrom[plugin.ChangeDetectionProvider](r)
	active := make([]plugin.ChangeDetectionProvider, 0, len(candidates))
	for _, c := range candidates {
		if isPluginEnabled(c) {
			active = append(active, c)
		}
	}
	return active
}

func detectorNames(detectors []plugin.ChangeDetectionProvider) string {
	var sb strings.Builder
	for i, d := range detectors {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(d.Name())
	}
	return sb.String()
}

func findProvider(candidates []ciProviderPlugin, name string) (*plugin.ResolvedCIProvider, error) {
	for _, c := range candidates {
		if c.ProviderName() == name {
			return buildResolvedCIProvider(c), nil
		}
	}
	return nil, fmt.Errorf("provider %q not found (available: %s)", name, providerNames(candidates))
}

func providerNames(candidates []ciProviderPlugin) string {
	var sb strings.Builder
	for i, c := range candidates {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(c.ProviderName())
	}
	return sb.String()
}

// ResolveKVCacheProvider returns a named KV cache backend provider.
func ResolveKVCacheProvider(name string) (plugin.KVCacheProvider, error) {
	return Default().ResolveKVCacheProvider(name)
}

// ResolveKVCacheProvider returns a named KV cache backend provider from this registry.
func (r *Registry) ResolveKVCacheProvider(name string) (plugin.KVCacheProvider, error) {
	if name == "" {
		return nil, errors.New("cache backend name is required")
	}

	resolved, ok := r.Get(name)
	if !ok {
		return nil, fmt.Errorf("cache backend %q not found", name)
	}

	provider, ok := resolved.(plugin.KVCacheProvider)
	if !ok {
		return nil, fmt.Errorf("plugin %q does not provide a KV cache backend", name)
	}
	if !isPluginEnabled(provider) {
		return nil, fmt.Errorf("cache backend %q is not active", name)
	}

	return provider, nil
}

// ResolveBlobStoreProvider returns a named blob store backend provider.
func ResolveBlobStoreProvider(name string) (plugin.BlobStoreProvider, error) {
	return Default().ResolveBlobStoreProvider(name)
}

// ResolveBlobStoreProvider returns a named blob store backend provider from this registry.
func (r *Registry) ResolveBlobStoreProvider(name string) (plugin.BlobStoreProvider, error) {
	if name == "" {
		return nil, errors.New("blob backend name is required")
	}

	resolved, ok := r.Get(name)
	if !ok {
		return nil, fmt.Errorf("blob backend %q not found", name)
	}

	provider, ok := resolved.(plugin.BlobStoreProvider)
	if !ok {
		return nil, fmt.Errorf("plugin %q does not provide a blob store backend", name)
	}
	if !isPluginEnabled(provider) {
		return nil, fmt.Errorf("blob backend %q is not active", name)
	}

	return provider, nil
}

// PreflightsForStartup returns enabled plugins that participate in framework
// preflight for the current config state.
func PreflightsForStartup() []plugin.Preflightable {
	return Default().PreflightsForStartup()
}

// PreflightsForStartup returns enabled plugins from this registry that
// participate in framework preflight for the current config state.
func (r *Registry) PreflightsForStartup() []plugin.Preflightable {
	plugins := r.All()
	result := make([]plugin.Preflightable, 0, len(plugins))
	for _, p := range plugins {
		if !isPluginEnabled(p) {
			continue
		}
		if preflightable, ok := p.(plugin.Preflightable); ok {
			result = append(result, preflightable)
		}
	}
	return result
}

// CollectContributions gathers pipeline contributions from all enabled
// PipelineContributor plugins.
func CollectContributions(ctx *plugin.AppContext) []*pipeline.Contribution {
	return Default().CollectContributions(ctx)
}

// CollectContributions gathers pipeline contributions from all enabled
// PipelineContributor plugins in this registry.
func (r *Registry) CollectContributions(ctx *plugin.AppContext) []*pipeline.Contribution {
	contributors := ByCapabilityFrom[plugin.PipelineContributor](r)
	contributions := make([]*pipeline.Contribution, 0, len(contributors))
	for _, c := range contributors {
		if cl, ok := c.(plugin.ConfigLoader); ok && !cl.IsEnabled() {
			continue
		}
		if contrib := c.PipelineContribution(ctx); contrib != nil {
			contributions = append(contributions, contrib)
		}
	}
	return contributions
}
