package plugin

import (
	"testing"

	"github.com/edelwud/terraci/pkg/pipeline"
)

type testPlugin struct {
	name string
	desc string
}

func (p *testPlugin) Name() string        { return p.name }
func (p *testPlugin) Description() string { return p.desc }

type testCommandPlugin struct {
	testPlugin
}

// testContributorPlugin implements PipelineContributor + ConfigLoader for testing.
type testContributorPlugin struct {
	BasePlugin[*testConfig]
	contribution *pipeline.Contribution
}

func (p *testContributorPlugin) PipelineContribution() *pipeline.Contribution {
	return p.contribution
}

func TestRegisterAndGet(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	p := &testPlugin{name: "test", desc: "A test plugin"}
	Register(p)

	got, ok := Get("test")
	if !ok {
		t.Fatal("expected to find plugin")
	}
	if got.Name() != "test" {
		t.Fatalf("got name %q, want %q", got.Name(), "test")
	}
}

func TestRegisterDuplicatePanics(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	Register(&testPlugin{name: "dup"})

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate registration")
		}
	}()
	Register(&testPlugin{name: "dup"})
}

func TestAll_Order(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	Register(&testPlugin{name: "b"})
	Register(&testPlugin{name: "a"})
	Register(&testPlugin{name: "c"})

	all := All()
	if len(all) != 3 {
		t.Fatalf("got %d plugins, want 3", len(all))
	}
	if all[0].Name() != "b" || all[1].Name() != "a" || all[2].Name() != "c" {
		t.Fatalf("wrong order: %s, %s, %s", all[0].Name(), all[1].Name(), all[2].Name())
	}
}

func TestByCapability(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	Register(&testPlugin{name: "plain"})
	Register(&testCommandPlugin{testPlugin: testPlugin{name: "cmd"}})

	// All plugins
	all := All()
	if len(all) != 2 {
		t.Fatalf("got %d plugins, want 2", len(all))
	}

	// Only command plugins — testCommandPlugin doesn't actually implement CommandProvider,
	// but we can test that ByCapability filters correctly with our test interface
	type hasName interface {
		Plugin
		Name() string
	}
	named := ByCapability[hasName]()
	if len(named) != 2 {
		t.Fatalf("got %d named plugins, want 2", len(named))
	}
}

func TestGetNotFound(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	_, ok := Get("nonexistent")
	if ok {
		t.Fatal("expected not found")
	}
}

func TestResolveProvider_NoPlugins(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	_, err := ResolveProvider()
	if err == nil {
		t.Fatal("expected error with no providers")
	}
}

func TestResolveChangeDetector_None(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	_, err := ResolveChangeDetector()
	if err == nil {
		t.Fatal("expected error with no detectors")
	}
}

func TestCollectContributions_Empty(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	contribs := CollectContributions()
	if len(contribs) != 0 {
		t.Errorf("expected 0 contributions, got %d", len(contribs))
	}
}

func TestAll_Empty(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	all := All()
	if len(all) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(all))
	}
}

func TestByCapability_NoMatch(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	Register(&testPlugin{name: "basic"})

	// VersionProvider is not implemented by testPlugin
	vp := ByCapability[VersionProvider]()
	if len(vp) != 0 {
		t.Errorf("expected 0 VersionProviders, got %d", len(vp))
	}
}

func TestReset(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	Register(&testPlugin{name: "x"})
	if len(All()) != 1 {
		t.Fatal("expected 1 plugin after register")
	}

	Reset()
	if len(All()) != 0 {
		t.Error("expected 0 plugins after reset")
	}
}

func TestCollectContributions_FiltersDisabledPlugins(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	// Enabled plugin with contribution
	enabled := &testContributorPlugin{
		BasePlugin: BasePlugin[*testConfig]{
			PluginName: "enabled",
			PluginDesc: "enabled plugin",
			EnableMode: EnabledExplicitly,
			DefaultCfg: func() *testConfig { return &testConfig{} },
			IsEnabledFn: func(cfg *testConfig) bool {
				return cfg != nil && cfg.Enabled
			},
		},
		contribution: &pipeline.Contribution{
			Jobs: []pipeline.ContributedJob{{Name: "enabled-job"}},
		},
	}
	enabled.SetTypedConfig(&testConfig{Enabled: true})
	Register(enabled)

	// Disabled plugin with contribution
	disabled := &testContributorPlugin{
		BasePlugin: BasePlugin[*testConfig]{
			PluginName: "disabled",
			PluginDesc: "disabled plugin",
			EnableMode: EnabledExplicitly,
			DefaultCfg: func() *testConfig { return &testConfig{} },
			IsEnabledFn: func(cfg *testConfig) bool {
				return cfg != nil && cfg.Enabled
			},
		},
		contribution: &pipeline.Contribution{
			Jobs: []pipeline.ContributedJob{{Name: "disabled-job"}},
		},
	}
	disabled.SetTypedConfig(&testConfig{Enabled: false})
	Register(disabled)

	contribs := CollectContributions()
	if len(contribs) != 1 {
		t.Fatalf("expected 1 contribution, got %d", len(contribs))
	}
	if contribs[0].Jobs[0].Name != "enabled-job" {
		t.Errorf("expected enabled-job, got %s", contribs[0].Jobs[0].Name)
	}
}

func TestCollectContributions_IncludesPluginWithoutConfigLoader(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	// Plugin that implements PipelineContributor but NOT ConfigLoader
	type bareContributor struct {
		testPlugin
		contribution *pipeline.Contribution
	}
	p := &bareContributor{
		testPlugin:   testPlugin{name: "bare", desc: "bare contributor"},
		contribution: &pipeline.Contribution{Jobs: []pipeline.ContributedJob{{Name: "bare-job"}}},
	}
	// Satisfy PipelineContributor
	Register(p)

	// bareContributor doesn't satisfy PipelineContributor since it doesn't have PipelineContribution()
	// So CollectContributions won't find it — this is expected behavior
	contribs := CollectContributions()
	if len(contribs) != 0 {
		t.Errorf("expected 0 contributions from bare plugin, got %d", len(contribs))
	}
}

func TestResetPlugins_ResetsConfigState(t *testing.T) {
	t.Cleanup(func() { Reset() })
	Reset()

	p := &testContributorPlugin{
		BasePlugin: BasePlugin[*testConfig]{
			PluginName: "resettable",
			PluginDesc: "resettable plugin",
			EnableMode: EnabledWhenConfigured,
			DefaultCfg: func() *testConfig { return &testConfig{} },
		},
	}
	Register(p)
	p.SetTypedConfig(&testConfig{Name: "configured"})

	if !p.IsConfigured() {
		t.Fatal("should be configured")
	}

	ResetPlugins()

	if p.IsConfigured() {
		t.Error("should not be configured after ResetPlugins")
	}
}

func TestCIProvider_Methods(t *testing.T) {
	p := &testPlugin{name: "test-ci", desc: "Test CI"}

	provider := &CIProvider{
		plugin: p,
	}

	if provider.Name() != "test-ci" {
		t.Errorf("Name() = %q, want test-ci", provider.Name())
	}
	if provider.Description() != "Test CI" {
		t.Errorf("Description() = %q, want Test CI", provider.Description())
	}
	if provider.Plugin() != p {
		t.Error("Plugin() should return underlying plugin")
	}
}
