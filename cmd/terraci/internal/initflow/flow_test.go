package initflow

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/initwiz"
	"github.com/edelwud/terraci/pkg/plugin/registry"

	_ "github.com/edelwud/terraci/plugins/github"
	_ "github.com/edelwud/terraci/plugins/gitlab"
)

type testSource struct {
	plugins []plugin.Plugin
}

func (s testSource) InitWizardSnapshot() (*registry.InitWizardSnapshot, error) {
	factories := make([]registry.Factory, 0, len(s.plugins))
	for _, p := range s.plugins {
		current := p
		factories = append(factories, func() plugin.Plugin { return current })
	}
	return registry.NewFromFactories(factories...).InitWizardSnapshot()
}

type testPlugin struct {
	name        string
	description string
}

func (p testPlugin) Name() string { return p.name }

func (p testPlugin) Description() string {
	if p.description != "" {
		return p.description
	}
	return p.name
}

type testProvider struct {
	testPlugin
	providerName string
}

func (p testProvider) ProviderName() string { return p.providerName }
func (p testProvider) PipelineID() string   { return "" }
func (p testProvider) CommitSHA() string    { return "" }

type testContributor struct {
	testPlugin
	groups   []initwiz.InitGroup
	groupErr error
	build    func(*initwiz.StateMap) (*initwiz.InitContribution, error)
}

func (p testContributor) InitGroups() ([]initwiz.InitGroup, error) {
	if p.groupErr != nil {
		return nil, p.groupErr
	}
	out := make([]initwiz.InitGroup, len(p.groups))
	for i := range p.groups {
		out[i] = p.groups[i].Clone()
	}
	return out, nil
}

func (p testContributor) BuildInitConfig(state *initwiz.StateMap) (*initwiz.InitContribution, error) {
	if p.build == nil {
		return nil, nil
	}
	return p.build(state)
}

type testConfig struct {
	Enabled bool   `yaml:"enabled,omitempty"`
	Value   string `yaml:"value,omitempty"`
}

func TestFlowDefaultStateProviderPreference(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		providers []plugin.Plugin
		want      string
	}{
		{
			name: "gitlab beats github",
			providers: []plugin.Plugin{
				testProvider{testPlugin: testPlugin{name: "zgithub"}, providerName: "github"},
				testProvider{testPlugin: testPlugin{name: "agitlab"}, providerName: "gitlab"},
			},
			want: "gitlab",
		},
		{
			name: "github beats lexical fallback",
			providers: []plugin.Plugin{
				testProvider{testPlugin: testPlugin{name: "azure"}, providerName: "azure"},
				testProvider{testPlugin: testPlugin{name: "github"}, providerName: "github"},
			},
			want: "github",
		},
		{
			name: "lexical fallback",
			providers: []plugin.Plugin{
				testProvider{testPlugin: testPlugin{name: "zeta"}, providerName: "zeta"},
				testProvider{testPlugin: testPlugin{name: "alpha"}, providerName: "alpha"},
			},
			want: "alpha",
		},
		{
			name:      "no provider",
			providers: nil,
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			flow := mustFlow(t, testSource{plugins: tt.providers})
			state := flow.DefaultState()

			if got := initwiz.ProviderKey.Get(state); got != tt.want {
				t.Fatalf("provider = %q, want %q", got, tt.want)
			}
			if got := initwiz.BinaryKey.Get(state); got != config.ExecutionBinaryTerraform {
				t.Fatalf("binary = %q, want terraform", got)
			}
			if got := initwiz.PatternKey.Get(state); got != config.Default().Structure().Pattern() {
				t.Fatalf("pattern = %q, want default pattern", got)
			}
			if got := initwiz.SummaryEnabledKey.Get(state); !got {
				t.Fatal("summary.enabled should default to true")
			}
		})
	}
}

func TestFlowApplyOverrides(t *testing.T) {
	t.Parallel()

	flow := mustFlow(t, testSource{})
	state := flow.DefaultState()

	flow.ApplyOverrides(state, Overrides{
		Provider: "github",
		Binary:   config.ExecutionBinaryTofu,
		Pattern:  "{environment}/{module}",
	})

	if got := initwiz.ProviderKey.Get(state); got != "github" {
		t.Fatalf("provider = %q", got)
	}
	if got := initwiz.BinaryKey.Get(state); got != config.ExecutionBinaryTofu {
		t.Fatalf("binary = %q", got)
	}
	if got := initwiz.PatternKey.Get(state); got != "{environment}/{module}" {
		t.Fatalf("pattern = %q", got)
	}
}

func TestFlowDisplayGroupsDeterministic(t *testing.T) {
	t.Parallel()

	left := testContributor{
		testPlugin: testPlugin{name: "left"},
		groups: []initwiz.InitGroup{
			groupSpec("Provider B", initwiz.CategoryProvider, 20, "provider.b"),
			groupSpec("Detail A", initwiz.CategoryDetail, 30, "detail.a"),
		},
	}
	right := testContributor{
		testPlugin: testPlugin{name: "right"},
		groups: []initwiz.InitGroup{
			groupSpec("Provider A", initwiz.CategoryProvider, 10, "provider.a"),
			groupSpec("Detail B", initwiz.CategoryDetail, 40, "detail.b"),
		},
	}

	first := mustFlow(t, testSource{plugins: []plugin.Plugin{right, left}}).DisplayGroups()
	second := mustFlow(t, testSource{plugins: []plugin.Plugin{left, right}}).DisplayGroups()

	if got, want := displayGroupSummary(first), displayGroupSummary(second); !reflect.DeepEqual(got, want) {
		t.Fatalf("display groups differ:\n got: %#v\nwant: %#v", got, want)
	}
	if got := displayGroupSummary(first); !reflect.DeepEqual(got, []string{
		"Provider A:provider.a",
		"Provider B:provider.b",
		"Detail A:detail.a",
		"Detail B:detail.b",
	}) {
		t.Fatalf("display group order = %#v", got)
	}
}

func TestFlowMergedGroupsDedupeFirstField(t *testing.T) {
	t.Parallel()

	contributor := testContributor{
		testPlugin: testPlugin{name: "features"},
		groups: []initwiz.InitGroup{
			mustGroup(t, initwiz.InitGroupOptions{
				Title:    "Beta",
				Category: initwiz.CategoryFeature,
				Order:    20,
				Fields: []initwiz.InitField{
					testStringField(t, "shared", "second"),
					testStringField(t, "beta", "beta"),
				},
			}),
			mustGroup(t, initwiz.InitGroupOptions{
				Title:    "Alpha",
				Category: initwiz.CategoryFeature,
				Order:    10,
				Fields: []initwiz.InitField{
					testStringField(t, "shared", "first"),
					testStringField(t, "alpha", "alpha"),
				},
			}),
		},
	}

	groups := mustFlow(t, testSource{plugins: []plugin.Plugin{contributor}}).DisplayGroups()
	if len(groups) != 1 {
		t.Fatalf("groups len = %d, want 1", len(groups))
	}
	if groups[0].Title() != "Features" {
		t.Fatalf("group title = %q", groups[0].Title())
	}
	fields := groups[0].Fields()
	got := fieldKeys(fields)
	want := []string{"shared", "alpha", "beta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("field keys = %#v, want %#v", got, want)
	}
	if got := fields[0].Title(); got != "first" {
		t.Fatalf("deduped field title = %q, want first", got)
	}
}

func TestFlowNewGroupError(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("bad group")
	_, err := New(testSource{plugins: []plugin.Plugin{
		testContributor{testPlugin: testPlugin{name: "bad"}, groupErr: errBoom},
	}})
	if err == nil {
		t.Fatal("New() error = nil")
	}
	var groupErr *registry.InitGroupError
	if !errors.As(err, &groupErr) {
		t.Fatalf("error %T does not wrap InitGroupError", err)
	}
	if groupErr.Plugin != "bad" {
		t.Fatalf("InitGroupError.Plugin = %q", groupErr.Plugin)
	}
	if !errors.Is(err, errBoom) {
		t.Fatalf("error does not wrap sentinel: %v", err)
	}
}

func TestFlowNewRejectsZeroValueGroup(t *testing.T) {
	t.Parallel()

	_, err := New(testSource{plugins: []plugin.Plugin{
		testContributor{testPlugin: testPlugin{name: "bad"}, groups: []initwiz.InitGroup{{}}},
	}})
	if err == nil {
		t.Fatal("New() error = nil")
	}
	var groupErr *registry.InitGroupError
	if !errors.As(err, &groupErr) {
		t.Fatalf("error %T does not wrap InitGroupError", err)
	}
	if !strings.Contains(err.Error(), "init group title is required") {
		t.Fatalf("error = %v, want invalid group message", err)
	}
}

func TestFlowBuildConfigContributionError(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("boom")
	contributor := testContributor{
		testPlugin: testPlugin{name: "bad"},
		build: func(*initwiz.StateMap) (*initwiz.InitContribution, error) {
			return nil, errBoom
		},
	}

	_, err := mustFlow(t, testSource{plugins: []plugin.Plugin{contributor}}).BuildConfig(nil)
	if err == nil {
		t.Fatal("BuildConfig() error = nil")
	}
	var contributionErr *registry.InitContributionError
	if !errors.As(err, &contributionErr) {
		t.Fatalf("error %T does not wrap InitContributionError", err)
	}
	if contributionErr.Plugin != "bad" {
		t.Fatalf("InitContributionError.Plugin = %q", contributionErr.Plugin)
	}
	if !errors.Is(err, errBoom) {
		t.Fatalf("error does not wrap sentinel: %v", err)
	}
}

func TestFlowBuildConfigDuplicateExtensionKeysFail(t *testing.T) {
	t.Parallel()

	build := func(*initwiz.StateMap) (*initwiz.InitContribution, error) {
		return initwiz.NewInitContribution(config.MustExtensionKey("dup"), testConfig{Enabled: true})
	}
	flow := mustFlow(t, testSource{plugins: []plugin.Plugin{
		testContributor{testPlugin: testPlugin{name: "a"}, build: build},
		testContributor{testPlugin: testPlugin{name: "b"}, build: build},
	}})

	_, err := flow.BuildConfig(nil)
	if err == nil {
		t.Fatal("BuildConfig() error = nil, want duplicate extension error")
	}
	if !strings.Contains(err.Error(), `duplicate extension "dup"`) {
		t.Fatalf("error = %v, want duplicate extension diagnostic", err)
	}
}

func TestFlowBuildConfigRealProvidersKeepCleanDefaults(t *testing.T) {
	t.Parallel()

	flow := mustFlow(t, registry.New())

	t.Run("gitlab", func(t *testing.T) {
		t.Parallel()

		state := flow.DefaultState()
		flow.ApplyOverrides(state, Overrides{Provider: "gitlab"})

		result, err := flow.BuildConfig(state)
		if err != nil {
			t.Fatalf("BuildConfig() error = %v", err)
		}
		data := marshalConfigYAML(t, result.Config)
		assertContains(t, data, "gitlab:")
		assertNotContains(t, data, "hashicorp/terraform:1.6")
		assertNotContains(t, data, "plan_mode:")
		assertNotContains(t, data, "plan_only:")
		assertNotContains(t, data, "cache_enabled:")
		assertContains(t, data, "cache:")
		assertContains(t, data, "enabled: true")
	})

	t.Run("github", func(t *testing.T) {
		t.Parallel()

		state := flow.DefaultState()
		flow.ApplyOverrides(state, Overrides{Provider: "github"})

		result, err := flow.BuildConfig(state)
		if err != nil {
			t.Fatalf("BuildConfig() error = %v", err)
		}
		data := marshalConfigYAML(t, result.Config)
		assertContains(t, data, "github:")
		assertContains(t, data, "runs_on: ubuntu-latest")
		assertNotContains(t, data, "plan_mode:")
		assertNotContains(t, data, "plan_only:")
	})
}

func mustFlow(t *testing.T, source PluginSource) *Flow {
	t.Helper()
	flow, err := New(source)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return flow
}

func groupSpec(title string, category initwiz.InitCategory, order int, fieldKey string) initwiz.InitGroup {
	field := testStringField(nil, fieldKey, fieldKey)
	group, err := initwiz.NewInitGroup(initwiz.InitGroupOptions{
		Title:    title,
		Category: category,
		Order:    order,
		Fields: []initwiz.InitField{
			field,
		},
	})
	if err != nil {
		panic(err)
	}
	return group
}

func mustGroup(t *testing.T, opts initwiz.InitGroupOptions) initwiz.InitGroup {
	t.Helper()
	group, err := initwiz.NewInitGroup(opts)
	if err != nil {
		t.Fatalf("NewInitGroup() error = %v", err)
	}
	return group
}

func testStringField(t *testing.T, key, title string) initwiz.InitField {
	if t != nil {
		t.Helper()
	}
	field, err := initwiz.NewStringField(initwiz.StringFieldOptions{
		Key:   initwiz.MustStateKey[string](key),
		Title: title,
	})
	if err != nil {
		if t != nil {
			t.Fatalf("NewStringField() error = %v", err)
		}
		panic(err)
	}
	return field
}

func displayGroupSummary(groups []DisplayGroup) []string {
	out := make([]string, 0, len(groups))
	for _, group := range groups {
		out = append(out, group.Title()+":"+strings.Join(fieldKeys(group.Fields()), ","))
	}
	return out
}

func fieldKeys(fields []initwiz.InitField) []string {
	out := make([]string, 0, len(fields))
	for i := range fields {
		out = append(out, fields[i].Key())
	}
	return out
}

func marshalConfigYAML(t *testing.T, cfg config.Config) string {
	t.Helper()

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}
	return string(data)
}

func assertContains(t *testing.T, text, needle string) {
	t.Helper()
	if !strings.Contains(text, needle) {
		t.Fatalf("text does not contain %q:\n%s", needle, text)
	}
}

func assertNotContains(t *testing.T, text, needle string) {
	t.Helper()
	if strings.Contains(text, needle) {
		t.Fatalf("text unexpectedly contains %q:\n%s", needle, text)
	}
}
