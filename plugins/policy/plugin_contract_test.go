package policy

import (
	"slices"
	"testing"

	"github.com/edelwud/terraci/pkg/plugin/initwiz"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

func TestPlugin_SDKContracts(t *testing.T) {
	t.Run("config", func(t *testing.T) {
		p := newTestPlugin()
		plugintest.AssertBaseConfigPlugin[*policyengine.Config](t, plugintest.BaseConfigPluginContract[*policyengine.Config]{
			Plugin:  p,
			Default: &policyengine.Config{},
			Configured: &policyengine.Config{
				Enabled:    true,
				Sources:    []policyengine.SourceConfig{{Type: policyengine.SourceTypePath, Path: "terraform"}},
				Namespaces: []string{"terraform"},
				Decisions:  policyengine.Decisions{Deny: policyengine.ActionBlock, Warn: policyengine.ActionWarn},
				Overrides: []policyengine.Override{{
					Match:      "prod/**",
					Namespaces: []string{"prod"},
					Decisions:  policyengine.Decisions{Deny: policyengine.ActionWarn},
				}},
			},
			Decoded: &policyengine.Config{
				Enabled:        true,
				Sources:        []policyengine.SourceConfig{{Type: policyengine.SourceTypeGit, URL: "https://example.invalid/policies.git", Ref: "main"}},
				Namespaces:     []string{"decoded"},
				Decisions:      policyengine.Decisions{Deny: policyengine.ActionWarn, Warn: policyengine.ActionIgnore},
				SourceCacheDir: ".terraci/policies",
			},
			Mutate: mutatePolicyConfig,
			Equal:  equalPolicyConfig,
		})
	})

	t.Run("command binding", func(t *testing.T) {
		p := newTestPlugin()
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
		enabled := newTestPlugin()
		enabled.SetTypedConfig(&policyengine.Config{Enabled: true})
		disabled := newTestPlugin()
		plugintest.AssertRequireEnabled(t, plugintest.RequireEnabledContract{
			Enabled:  enabled,
			Disabled: disabled,
			Message:  "policy disabled",
		})
	})

	t.Run("preflight", func(t *testing.T) {
		p := newTestPlugin()
		p.SetTypedConfig(&policyengine.Config{
			Enabled:   true,
			Sources:   []policyengine.SourceConfig{{Type: policyengine.SourceTypePath, Path: "policies"}},
			Decisions: policyengine.Decisions{Deny: policyengine.ActionWarn},
		})
		plugintest.AssertPreflightable(t, plugintest.PreflightableContract{
			Plugin:     p,
			AppContext: plugintest.NewAppContext(t, t.TempDir()),
		})
	})

	t.Run("init contributor", func(t *testing.T) {
		state := initwiz.NewStateMap()
		state.Set("policy.enabled", true)
		plugintest.AssertInitContributor(t, plugintest.InitContributorContract{
			Contributor:        newTestPlugin(),
			State:              state,
			ExpectedPluginKey:  pluginName,
			ExpectContribution: true,
			DecodeTarget:       &policyengine.Config{},
		})
	})

	t.Run("version provider", func(t *testing.T) {
		plugintest.AssertVersionProvider(t, plugintest.VersionProviderContract{
			Provider:     newTestPlugin(),
			ExpectedKeys: []string{"opa"},
		})
	})
}

func mutatePolicyConfig(c *policyengine.Config) {
	if c == nil {
		return
	}
	c.Enabled = !c.Enabled
	c.Sources = append(c.Sources, policyengine.SourceConfig{Type: policyengine.SourceTypeOCI, URL: "oci://mutated"})
	c.Namespaces = append(c.Namespaces, "mutated")
	c.SourceCacheDir = "mutated"
	if len(c.Overrides) > 0 {
		c.Overrides[0].Namespaces = append(c.Overrides[0].Namespaces, "mutated")
	}
}

func equalPolicyConfig(got, want *policyengine.Config) bool {
	if got == nil || want == nil {
		return got == want
	}
	return got.Enabled == want.Enabled &&
		got.Decisions == want.Decisions &&
		got.SourceCacheDir == want.SourceCacheDir &&
		slices.Equal(got.Sources, want.Sources) &&
		slices.Equal(got.Namespaces, want.Namespaces) &&
		slices.EqualFunc(got.Overrides, want.Overrides, equalPolicyOverride)
}

func equalPolicyOverride(got, want policyengine.Override) bool {
	return got.Match == want.Match &&
		enabledPointerEqual(got.Enabled, want.Enabled) &&
		got.Decisions == want.Decisions &&
		slices.Equal(got.Namespaces, want.Namespaces)
}

func enabledPointerEqual(got, want *bool) bool {
	if got == nil || want == nil {
		return got == want
	}
	return *got == *want
}
