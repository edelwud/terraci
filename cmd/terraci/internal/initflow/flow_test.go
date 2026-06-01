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

func (s testSource) All() []plugin.Plugin {
	return append([]plugin.Plugin(nil), s.plugins...)
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
	groups []*initwiz.InitGroupSpec
	build  func(*initwiz.StateMap) (*initwiz.InitContribution, error)
}

func (p testContributor) InitGroups() []*initwiz.InitGroupSpec {
	return append([]*initwiz.InitGroupSpec(nil), p.groups...)
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

			flow := New(testSource{plugins: tt.providers})
			state := flow.DefaultState()

			if got := initwiz.ProviderKey.Get(state); got != tt.want {
				t.Fatalf("provider = %q, want %q", got, tt.want)
			}
			if got := initwiz.BinaryKey.Get(state); got != config.ExecutionBinaryTerraform {
				t.Fatalf("binary = %q, want terraform", got)
			}
			if got := initwiz.PatternKey.Get(state); got != config.DefaultConfig().Structure.Pattern {
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

	flow := New(testSource{})
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
		groups: []*initwiz.InitGroupSpec{
			groupSpec("Provider B", initwiz.CategoryProvider, 20, "provider.b"),
			groupSpec("Detail A", initwiz.CategoryDetail, 30, "detail.a"),
		},
	}
	right := testContributor{
		testPlugin: testPlugin{name: "right"},
		groups: []*initwiz.InitGroupSpec{
			groupSpec("Provider A", initwiz.CategoryProvider, 10, "provider.a"),
			groupSpec("Detail B", initwiz.CategoryDetail, 40, "detail.b"),
		},
	}

	first := New(testSource{plugins: []plugin.Plugin{right, left}}).DisplayGroups()
	second := New(testSource{plugins: []plugin.Plugin{left, right}}).DisplayGroups()

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
		groups: []*initwiz.InitGroupSpec{
			{
				Title:    "Beta",
				Category: initwiz.CategoryFeature,
				Order:    20,
				Fields: []initwiz.InitField{
					testStringField("shared", "second"),
					testStringField("beta", "beta"),
				},
			},
			{
				Title:    "Alpha",
				Category: initwiz.CategoryFeature,
				Order:    10,
				Fields: []initwiz.InitField{
					testStringField("shared", "first"),
					testStringField("alpha", "alpha"),
				},
			},
		},
	}

	groups := New(testSource{plugins: []plugin.Plugin{contributor}}).DisplayGroups()
	if len(groups) != 1 {
		t.Fatalf("groups len = %d, want 1", len(groups))
	}
	if groups[0].Title != "Features" {
		t.Fatalf("group title = %q", groups[0].Title)
	}
	got := fieldKeys(groups[0].Fields)
	want := []string{"shared", "alpha", "beta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("field keys = %#v, want %#v", got, want)
	}
	if got := groups[0].Fields[0].Title(); got != "first" {
		t.Fatalf("deduped field title = %q, want first", got)
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

	_, err := New(testSource{plugins: []plugin.Plugin{contributor}}).BuildConfig(nil)
	if err == nil {
		t.Fatal("BuildConfig() error = nil")
	}
	var contributionErr *ContributionError
	if !errors.As(err, &contributionErr) {
		t.Fatalf("error %T does not wrap ContributionError", err)
	}
	if contributionErr.Plugin != "bad" {
		t.Fatalf("ContributionError.Plugin = %q", contributionErr.Plugin)
	}
	if !errors.Is(err, errBoom) {
		t.Fatalf("error does not wrap sentinel: %v", err)
	}
}

func TestFlowBuildConfigDuplicateExtensionKeysFail(t *testing.T) {
	t.Parallel()

	build := func(*initwiz.StateMap) (*initwiz.InitContribution, error) {
		return initwiz.NewInitContribution("dup", testConfig{Enabled: true})
	}
	flow := New(testSource{plugins: []plugin.Plugin{
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

	flow := New(registry.New())

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

func groupSpec(title string, category initwiz.InitCategory, order int, fieldKey string) *initwiz.InitGroupSpec {
	return &initwiz.InitGroupSpec{
		Title:    title,
		Category: category,
		Order:    order,
		Fields: []initwiz.InitField{
			testStringField(fieldKey, fieldKey),
		},
	}
}

func testStringField(key, title string) initwiz.InitField {
	return initwiz.NewStringField(initwiz.StringFieldOptions{
		Key:   initwiz.MustStateKey[string](key),
		Title: title,
	})
}

func displayGroupSummary(groups []DisplayGroup) []string {
	out := make([]string, 0, len(groups))
	for _, group := range groups {
		out = append(out, group.Title+":"+strings.Join(fieldKeys(group.Fields), ","))
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

func marshalConfigYAML(t *testing.T, cfg *config.Config) string {
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
