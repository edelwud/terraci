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
// CommentFactory is optional — checked via type assertion in buildCIProvider.
type ciProviderPlugin interface {
	plugin.Plugin
	plugin.EnvDetector
	plugin.CIMetadata
	plugin.GeneratorFactory
}

func activeCIProviders() []ciProviderPlugin {
	candidates := ByCapability[ciProviderPlugin]()
	active := make([]ciProviderPlugin, 0, len(candidates))
	for _, c := range candidates {
		if isPluginEnabled(c) {
			active = append(active, c)
		}
	}
	return active
}

// ResolveProvider detects the active CI provider.
// Priority: env detection → TERRACI_PROVIDER env → single registered → configured.
func ResolveProvider() (*plugin.CIProvider, error) {
	candidates := activeCIProviders()
	if len(candidates) == 0 {
		return nil, errors.New("no active CI provider plugins registered")
	}

	// Check env detection (CI environment variables)
	for _, c := range candidates {
		if c.DetectEnv() {
			return buildCIProvider(c), nil
		}
	}

	// Check TERRACI_PROVIDER env var
	if name := os.Getenv("TERRACI_PROVIDER"); name != "" {
		return findProvider(candidates, name)
	}

	// Single provider registered
	if len(candidates) == 1 {
		return buildCIProvider(candidates[0]), nil
	}

	return nil, fmt.Errorf("cannot determine CI provider: multiple plugins registered (%s), set TERRACI_PROVIDER", providerNames(candidates))
}

func buildCIProvider(p ciProviderPlugin) *plugin.CIProvider {
	var comment plugin.CommentFactory
	if cf, ok := p.(plugin.CommentFactory); ok {
		comment = cf
	}
	return plugin.NewCIProvider(p, p, p, comment)
}

// ResolveChangeDetector returns the active ChangeDetectionProvider.
// Priority: single registered → configured+enabled → error.
func ResolveChangeDetector() (plugin.ChangeDetectionProvider, error) {
	detectors := ByCapability[plugin.ChangeDetectionProvider]()
	if len(detectors) == 0 {
		return nil, errors.New("no change detection plugin registered")
	}
	if len(detectors) == 1 {
		return detectors[0], nil
	}
	for _, d := range detectors {
		if cl, ok := d.(plugin.ConfigLoader); ok && cl.IsEnabled() {
			return d, nil
		}
	}
	return nil, fmt.Errorf("cannot determine change detector: multiple plugins registered (%s)",
		detectorNames(detectors))
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

func findProvider(candidates []ciProviderPlugin, name string) (*plugin.CIProvider, error) {
	for _, c := range candidates {
		if c.ProviderName() == name {
			return buildCIProvider(c), nil
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

// PreflightsForStartup returns enabled plugins that participate in framework
// preflight for the current config state.
func PreflightsForStartup() []plugin.Preflightable {
	result := make([]plugin.Preflightable, 0, len(All()))
	for _, p := range All() {
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
	contributors := ByCapability[plugin.PipelineContributor]()
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
