package summary

import (
	"slices"
	"testing"

	"github.com/edelwud/terraci/pkg/plugin/initwiz"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	summaryengine "github.com/edelwud/terraci/plugins/summary/internal/summaryengine"
)

func TestPlugin_SDKContracts(t *testing.T) {
	t.Run("config", func(t *testing.T) {
		p := newTestPlugin()
		enabled := true
		disabled := false
		include := true
		includeDecoded := false
		plugintest.AssertBaseConfigPlugin[*summaryengine.Config](t, plugintest.BaseConfigPluginContract[*summaryengine.Config]{
			Plugin:     p,
			Default:    &summaryengine.Config{},
			Configured: &summaryengine.Config{Enabled: &enabled, IncludeDetails: &include, Labels: []string{"terraform", "{module}"}},
			Decoded:    &summaryengine.Config{Enabled: &disabled, IncludeDetails: &includeDecoded, Labels: []string{"decoded"}},
			Mutate:     mutateSummaryConfig,
			Equal:      equalSummaryConfig,
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
		disabled := newTestPlugin()
		off := false
		disabled.SetTypedConfig(&summaryengine.Config{Enabled: &off})
		plugintest.AssertRequireEnabled(t, plugintest.RequireEnabledContract{
			Enabled:  enabled,
			Disabled: disabled,
			Message:  "summary disabled",
		})
	})

	t.Run("pipeline contribution", func(t *testing.T) {
		plugintest.AssertPipelineContributor(t, plugintest.PipelineContributorContract{
			Contributor:      newTestPlugin(),
			AppContext:       plugintest.NewAppContext(t, t.TempDir()),
			ExpectedJobNames: []string{"terraci-summary"},
		})
	})

	t.Run("init contributor", func(t *testing.T) {
		state := initwiz.NewStateMap()
		state.Set("summary.enabled", false)
		plugintest.AssertInitContributor(t, plugintest.InitContributorContract{
			Contributor:        newTestPlugin(),
			State:              state,
			ExpectedPluginKey:  pluginName,
			ExpectContribution: true,
			DecodeTarget:       &summaryengine.Config{},
		})
	})
}

func mutateSummaryConfig(c *summaryengine.Config) {
	if c == nil {
		return
	}
	if c.Enabled != nil {
		next := !*c.Enabled
		c.Enabled = &next
	}
	if c.IncludeDetails != nil {
		next := !*c.IncludeDetails
		c.IncludeDetails = &next
	}
	c.Labels = append(c.Labels, "mutated")
}

func equalSummaryConfig(got, want *summaryengine.Config) bool {
	if got == nil || want == nil {
		return got == want
	}
	return boolPointerEqual(got.Enabled, want.Enabled) &&
		boolPointerEqual(got.IncludeDetails, want.IncludeDetails) &&
		got.OnChangesOnly == want.OnChangesOnly &&
		slices.Equal(got.Labels, want.Labels)
}

func boolPointerEqual(got, want *bool) bool {
	if got == nil || want == nil {
		return got == want
	}
	return *got == *want
}
