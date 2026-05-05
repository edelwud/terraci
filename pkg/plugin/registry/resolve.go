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
// CommentServiceFactory is optional — checked via type assertion in
// buildResolvedCIProvider.
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

// ResolveCIProvider detects the active CI provider in this registry.
// Priority: TERRACI_PROVIDER env → CI environment detection → single active
// provider → error.
func (r *Registry) ResolveCIProvider() (*plugin.ResolvedCIProvider, error) {
	candidates := r.activeCIProviders()
	if len(candidates) == 0 {
		return nil, errors.New("no active CI provider plugins — configure extensions.gitlab or extensions.github in .terraci.yaml")
	}

	// Explicit selection wins over auto-detection. This is important for local
	// debugging inside CI-like environments and keeps CLI/env overrides predictable.
	if name := os.Getenv("TERRACI_PROVIDER"); name != "" {
		return findProvider(candidates, name)
	}

	for _, c := range candidates {
		if c.DetectEnv() {
			return buildResolvedCIProvider(c), nil
		}
	}

	if len(candidates) == 1 {
		return buildResolvedCIProvider(candidates[0]), nil
	}

	// Multiple providers configured but none auto-detected — emit an
	// actionable error that lists the candidates and the resolution
	// priority so the user knows which knob to turn.
	names := providerNames(candidates)
	return nil, fmt.Errorf(
		"cannot determine CI provider: %d providers configured (%s) but none auto-detected from CI environment. "+
			"Resolution priority: 1) TERRACI_PROVIDER env, 2) DetectEnv() match, 3) single configured provider. "+
			"Set TERRACI_PROVIDER=<name> or configure only one of [%s] in .terraci.yaml",
		len(candidates), names, names,
	)
}

func buildResolvedCIProvider(p ciProviderPlugin) *plugin.ResolvedCIProvider {
	var comment plugin.CommentServiceFactory
	if cf, ok := p.(plugin.CommentServiceFactory); ok {
		comment = cf
	}
	return plugin.NewResolvedCIProvider(p, p, p, comment)
}

// ResolveChangeDetector returns the single active ChangeDetectionProvider in
// this registry.
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

// ResolveKVCacheProvider returns a named KV cache backend provider. When name
// is empty, falls back to the single enabled KV cache provider (mirrors the
// "single active provider" path in ResolveCIProvider) — otherwise returns an
// error listing the available backends so the caller can disambiguate.
func (r *Registry) ResolveKVCacheProvider(name string) (plugin.KVCacheProvider, error) {
	if name == "" {
		return r.singleActiveKVCacheProvider()
	}

	resolved, ok := r.GetPlugin(name)
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

// ResolveBlobStoreProvider returns a named blob store backend provider. When
// name is empty, falls back to the single enabled blob store provider —
// otherwise returns an error listing the available backends.
func (r *Registry) ResolveBlobStoreProvider(name string) (plugin.BlobStoreProvider, error) {
	if name == "" {
		return r.singleActiveBlobStoreProvider()
	}

	resolved, ok := r.GetPlugin(name)
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

func (r *Registry) singleActiveKVCacheProvider() (plugin.KVCacheProvider, error) {
	candidates := ByCapabilityFrom[plugin.KVCacheProvider](r)
	active := make([]plugin.KVCacheProvider, 0, len(candidates))
	for _, c := range candidates {
		if isPluginEnabled(c) {
			active = append(active, c)
		}
	}
	switch len(active) {
	case 0:
		return nil, errors.New("no active KV cache provider — set extensions.<feature>.cache.backend explicitly")
	case 1:
		return active[0], nil
	default:
		return nil, fmt.Errorf("multiple active KV cache providers (%s) — set extensions.<feature>.cache.backend explicitly",
			pluginNames(active))
	}
}

func (r *Registry) singleActiveBlobStoreProvider() (plugin.BlobStoreProvider, error) {
	candidates := ByCapabilityFrom[plugin.BlobStoreProvider](r)
	active := make([]plugin.BlobStoreProvider, 0, len(candidates))
	for _, c := range candidates {
		if isPluginEnabled(c) {
			active = append(active, c)
		}
	}
	switch len(active) {
	case 0:
		return nil, errors.New("no active blob store provider — set extensions.<feature>.blob_cache.backend explicitly")
	case 1:
		return active[0], nil
	default:
		return nil, fmt.Errorf("multiple active blob store providers (%s) — set extensions.<feature>.blob_cache.backend explicitly",
			pluginNames(active))
	}
}

// pluginNames returns a comma-separated list of plugin names from any slice
// whose elements satisfy the Plugin interface.
func pluginNames[T plugin.Plugin](items []T) string {
	var sb strings.Builder
	for i, item := range items {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(item.Name())
	}
	return sb.String()
}

// PreflightsForStartup returns enabled plugins that participate in framework
// preflight for the current configuration state.
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
