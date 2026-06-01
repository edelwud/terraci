package tfupdate

import (
	"maps"
	"slices"
	"testing"

	"github.com/edelwud/terraci/pkg/plugin/initwiz"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	tfupdateengine "github.com/edelwud/terraci/plugins/tfupdate/internal"
)

func TestPlugin_SDKContracts(t *testing.T) {
	t.Run("config", func(t *testing.T) {
		p := newTestPlugin(t)
		plugintest.AssertBaseConfigPlugin[*tfupdateengine.UpdateConfig](t, plugintest.BaseConfigPluginContract[*tfupdateengine.UpdateConfig]{
			Plugin: p,
			Default: &tfupdateengine.UpdateConfig{
				Target: tfupdateengine.TargetAll,
				Policy: tfupdateengine.UpdatePolicy{Bump: tfupdateengine.BumpMinor},
			},
			Configured: &tfupdateengine.UpdateConfig{
				Enabled: true,
				Target:  tfupdateengine.TargetProviders,
				Ignore:  []string{"hashicorp/null"},
				Policy:  tfupdateengine.UpdatePolicy{Bump: tfupdateengine.BumpPatch, Pin: true},
				Registries: tfupdateengine.RegistryConfig{
					Default:   "registry.terraform.io",
					Providers: map[string]string{"hashicorp/aws": "registry.terraform.io"},
				},
				Lock: tfupdateengine.LockConfig{Platforms: []string{"linux_amd64"}},
				Cache: &tfupdateengine.CacheConfig{
					Metadata:  tfupdateengine.MetadataCacheConfig{Backend: "inmemcache", Namespace: "metadata", TTL: "1h"},
					Artifacts: tfupdateengine.ArtifactCacheConfig{Backend: "diskblob", Namespace: "artifacts"},
				},
			},
			Decoded: &tfupdateengine.UpdateConfig{
				Enabled: true,
				Target:  tfupdateengine.TargetModules,
				Ignore:  []string{"terraform-aws-modules/vpc/aws"},
				Policy:  tfupdateengine.UpdatePolicy{Bump: tfupdateengine.BumpMajor},
				Registries: tfupdateengine.RegistryConfig{
					Default:   "private.example",
					Providers: map[string]string{"hashicorp/random": "private.example"},
				},
				Lock: tfupdateengine.LockConfig{Platforms: []string{"darwin_arm64"}},
			},
			Mutate: mutateUpdateConfig,
			Equal:  equalUpdateConfig,
		})
	})

	t.Run("command binding", func(t *testing.T) {
		p := newTestPlugin(t)
		plugintest.AssertCommandBinding[*Plugin](t, plugintest.CommandBindingContract[*Plugin]{
			Name:   pluginName,
			Plugin: p,
			AssertResolved: func(tb testing.TB, got *Plugin) {
				tb.Helper()
				if got != p {
					tb.Fatalf("resolved plugin = %p, want %p", got, p)
				}
			},
		})
	})

	t.Run("require enabled", func(t *testing.T) {
		enabled := newTestPlugin(t)
		enablePlugin(t, enabled, &tfupdateengine.UpdateConfig{Enabled: true})
		disabled := newTestPlugin(t)
		plugintest.AssertRequireEnabled(t, plugintest.RequireEnabledContract{
			Enabled:  enabled,
			Disabled: disabled,
			Message:  "tfupdate disabled",
		})
	})

	t.Run("preflight", func(t *testing.T) {
		p := newTestPlugin(t)
		enablePlugin(t, p, &tfupdateengine.UpdateConfig{Enabled: true})
		plugintest.AssertPreflightable(t, plugintest.PreflightableContract{
			Plugin:     p,
			AppContext: newTestAppContext(t, t.TempDir()),
		})
	})

	t.Run("init contributor", func(t *testing.T) {
		state := initwiz.NewStateMap()
		keyUpdateEnabled.Set(state, true)
		plugintest.AssertInitContributor(t, plugintest.InitContributorContract{
			Contributor:        newTestPlugin(t),
			State:              state,
			ExpectedPluginKey:  pluginName,
			ExpectContribution: true,
			DecodeTarget:       &tfupdateengine.UpdateConfig{},
		})
	})
}

func mutateUpdateConfig(c *tfupdateengine.UpdateConfig) {
	if c == nil {
		return
	}
	c.Enabled = !c.Enabled
	c.Ignore = append(c.Ignore, "mutated")
	if c.Registries.Providers == nil {
		c.Registries.Providers = map[string]string{}
	}
	c.Registries.Providers["mutated"] = "registry.invalid"
	c.Lock.Platforms = append(c.Lock.Platforms, "mutated_platform")
	if c.Cache != nil {
		c.Cache.Metadata.Namespace = "mutated"
	}
}

func equalUpdateConfig(got, want *tfupdateengine.UpdateConfig) bool {
	if got == nil || want == nil {
		return got == want
	}
	return got.Enabled == want.Enabled &&
		got.Target == want.Target &&
		got.Pipeline == want.Pipeline &&
		got.Timeout == want.Timeout &&
		got.Policy == want.Policy &&
		got.Registries.Default == want.Registries.Default &&
		maps.Equal(got.Registries.Providers, want.Registries.Providers) &&
		slices.Equal(got.Ignore, want.Ignore) &&
		slices.Equal(got.Lock.Platforms, want.Lock.Platforms) &&
		equalUpdateCache(got.Cache, want.Cache)
}

func equalUpdateCache(got, want *tfupdateengine.CacheConfig) bool {
	if got == nil || want == nil {
		return got == want
	}
	return got.Metadata == want.Metadata && got.Artifacts == want.Artifacts
}
