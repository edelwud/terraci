package registry

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
)

// activeByCapability returns plugins from r that implement T and are
// currently enabled. It replaces what used to be four near-identical
// "fetch-and-filter" helpers (one per capability).
func activeByCapability[T plugin.Plugin](r *Registry) []T {
	candidates := ByCapabilityFrom[T](r)
	active := candidates[:0]
	for _, c := range candidates {
		if isPluginEnabled(c) {
			active = append(active, c)
		}
	}
	return active
}

// resolveSingle returns the only enabled candidate or maps the {0, many}
// cases onto descriptive errors. kind is the human-friendly capability name
// used to compose error messages (e.g. "KV cache provider").
func resolveSingle[T plugin.Plugin](active []T, kind, hint string) (T, error) {
	var zero T
	switch len(active) {
	case 0:
		if hint != "" {
			return zero, fmt.Errorf("no active %s — %s", kind, hint)
		}
		return zero, fmt.Errorf("no active %s", kind)
	case 1:
		return active[0], nil
	default:
		if hint != "" {
			return zero, fmt.Errorf("multiple active %ss (%s) — %s", kind, pluginNames(active), hint)
		}
		return zero, fmt.Errorf("multiple active %ss (%s)", kind, pluginNames(active))
	}
}

// resolveNamedBackend returns a named provider, falling back to the single
// enabled candidate when name is empty. Used for KV cache and blob store
// resolution where the user names a backend in config.
func resolveNamedBackend[T plugin.Plugin](r *Registry, name, kind, hint string) (T, error) {
	var zero T
	if name == "" {
		return resolveSingle(activeByCapability[T](r), kind, hint)
	}

	resolved, ok := r.GetPlugin(name)
	if !ok {
		return zero, fmt.Errorf("%s %q not found", kind, name)
	}

	provider, ok := resolved.(T)
	if !ok {
		return zero, fmt.Errorf("plugin %q does not provide a %s", name, kind)
	}
	if !isPluginEnabled(provider) {
		return zero, fmt.Errorf("%s %q is not active", kind, name)
	}

	return provider, nil
}

// ciProviderPlugin is the minimum interface set for a CI provider plugin.
// CommentServiceFactory is optional — checked via type assertion in
// buildResolvedCIProvider.
type ciProviderPlugin interface {
	plugin.Plugin
	plugin.EnvDetector
	plugin.CIInfoProvider
	plugin.PipelineGeneratorFactory
}

// ResolveCIProvider detects the active CI provider in this registry.
// Priority: TERRACI_PROVIDER env → CI environment detection → single active
// provider → error.
func (r *Registry) ResolveCIProvider() (*plugin.ResolvedCIProvider, error) {
	candidates := activeByCapability[ciProviderPlugin](r)
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

// ResolveChangeDetector returns the single active ChangeDetectionProvider.
func (r *Registry) ResolveChangeDetector() (plugin.ChangeDetectionProvider, error) {
	return resolveSingle(
		activeByCapability[plugin.ChangeDetectionProvider](r),
		"change detector",
		"",
	)
}

// ResolveKVCacheProvider returns a named KV cache backend provider. When
// name is empty, falls back to the single enabled KV cache provider —
// otherwise returns an error pointing at the available backends.
func (r *Registry) ResolveKVCacheProvider(name string, configPathHint ...string) (plugin.KVCacheProvider, error) {
	return resolveNamedBackend[plugin.KVCacheProvider](
		r, name,
		"cache backend",
		firstHint(configPathHint, "set the feature cache backend explicitly"),
	)
}

// ResolveBlobStoreProvider returns a named blob store backend provider.
// When name is empty, falls back to the single enabled blob store provider.
func (r *Registry) ResolveBlobStoreProvider(name string, configPathHint ...string) (plugin.BlobStoreProvider, error) {
	return resolveNamedBackend[plugin.BlobStoreProvider](
		r, name,
		"blob backend",
		firstHint(configPathHint, "set the feature blob backend explicitly"),
	)
}

func firstHint(hints []string, fallback string) string {
	for _, hint := range hints {
		if hint != "" {
			return hint
		}
	}
	return fallback
}

// pluginNames returns a comma-separated list of plugin names from any
// slice whose elements satisfy the Plugin interface.
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

// PreflightsForStartup returns enabled plugins that participate in
// framework preflight for the current configuration state.
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
