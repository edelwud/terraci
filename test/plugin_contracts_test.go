package test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
)

func TestBuiltInPluginContractMatrix(t *testing.T) {
	expected := map[string]struct {
		configLoader bool
		command      bool
		preflight    bool
		pipeline     bool
	}{
		"cost": {
			configLoader: true,
			command:      true,
			preflight:    true,
			pipeline:     true,
		},
		"diskblob": {
			configLoader: true,
		},
		"git": {
			preflight: true,
		},
		"github": {
			configLoader: true,
			preflight:    true,
		},
		"gitlab": {
			configLoader: true,
			preflight:    true,
		},
		"inmemcache": {
			configLoader: true,
		},
		"local-exec": {
			command: true,
		},
		"policy": {
			configLoader: true,
			command:      true,
			preflight:    true,
			pipeline:     true,
		},
		"summary": {
			configLoader: true,
			command:      true,
			pipeline:     true,
		},
		"tfupdate": {
			configLoader: true,
			command:      true,
			preflight:    true,
			pipeline:     true,
		},
	}

	for _, p := range registry.New().Inventory().Plugins() {
		want, ok := expected[p.Name()]
		if !ok {
			t.Fatalf("unexpected plugin %q in registry", p.Name())
		}

		if got := p.HasConfigLoader(); got != want.configLoader {
			t.Errorf("%s ConfigLoader = %v, want %v", p.Name(), got, want.configLoader)
		}
		if got := p.HasCommandProvider(); got != want.command {
			t.Errorf("%s CommandProvider = %v, want %v", p.Name(), got, want.command)
		}
		if got := p.HasPreflight(); got != want.preflight {
			t.Errorf("%s Preflightable = %v, want %v", p.Name(), got, want.preflight)
		}
		if got := p.HasPipelineContributor(); got != want.pipeline {
			t.Errorf("%s PipelineContributor = %v, want %v", p.Name(), got, want.pipeline)
		}
	}
}

func TestGeneratedSchemaExcludesGitExtension(t *testing.T) {
	plugins := registry.New()

	schema := generatedSchema(t, plugins)
	if strings.Contains(schema, `"git":`) {
		t.Fatalf("generated schema unexpectedly contains extensions.git: %s", schema)
	}
}

func TestGeneratedSchemaUsesCanonicalPolicyFields(t *testing.T) {
	plugins := registry.New()

	schema := generatedSchema(t, plugins)
	for _, removed := range []string{"failure_action", "warning_action", "cache_dir"} {
		if strings.Contains(schema, `"`+removed+`"`) {
			t.Fatalf("generated schema contains legacy policy field %q: %s", removed, schema)
		}
	}
	for _, want := range []string{"decisions", "source_cache_dir"} {
		if !strings.Contains(schema, want) {
			t.Fatalf("generated schema missing policy field %q: %s", want, schema)
		}
	}
}

func TestGeneratedSchemaIncludesSummaryFields(t *testing.T) {
	plugins := registry.New()

	schema := generatedSchema(t, plugins)
	for _, want := range []string{"enabled", "on_changes_only", "include_details", "labels"} {
		if !strings.Contains(schema, want) {
			t.Fatalf("generated schema missing summary field %q: %s", want, schema)
		}
	}
}

func generatedSchema(tb testing.TB, plugins *registry.Registry) string {
	tb.Helper()
	definitions, err := plugins.ExtensionDefinitions()
	if err != nil {
		tb.Fatalf("ExtensionDefinitions() error = %v", err)
	}
	schema, err := config.GenerateJSONSchema(definitions)
	if err != nil {
		tb.Fatalf("GenerateJSONSchema() error = %v", err)
	}
	return schema
}

func TestRunPreflight_UsesEnabledPlugins(t *testing.T) {
	appCtx, plugins := loadPluginContractConfig(t, `service_dir: .terraci
structure:
  pattern: "{service}/{environment}/{region}/{module}"
execution:
  binary: terraform
extensions:
  gitlab:
    image:
      name: hashicorp/terraform:1.6
  cost:
    providers:
      aws:
        enabled: true
  policy:
    enabled: true
    sources:
      - type: path
        path: terraform
  summary: {}
  tfupdate:
    enabled: true
`)

	if err := plugins.RunPreflight(context.Background(), appCtx); err != nil {
		if !strings.Contains(err.Error(), "preflight plugin git") {
			t.Fatalf("RunPreflight() error = %v", err)
		}
	}
}

func TestCollectContributions_UsesContextualStateWithoutPreflight(t *testing.T) {
	appCtx, plugins := loadPluginContractConfig(t, `service_dir: custom-artifacts
structure:
  pattern: "{service}/{environment}/{region}/{module}"
extensions:
  cost:
    providers:
      aws:
        enabled: true
  policy:
    enabled: true
    sources:
      - type: path
        path: terraform
  summary: {}
  tfupdate:
    enabled: true
    pipeline: true
`)

	contributions, err := plugins.CollectContributions(appCtx)
	if err != nil {
		t.Fatalf("CollectContributions() error = %v", err)
	}
	if contributions.Len() != 4 {
		t.Fatalf("CollectContributions() returned %d contributions, want 4", contributions.Len())
	}

	foundUpdateArtifactPath := false
	for _, contrib := range contributions.Contributions() {
		for _, job := range contrib.Jobs() {
			if job.Name() != "tfupdate-check" {
				continue
			}
			produces := job.Produces()
			if len(produces) != 2 {
				t.Fatalf("tfupdate-check produces = %#v, want result and report", produces)
			}
			wantPaths := []string{
				pipeline.WorkspacePath("custom-artifacts", ci.ResultFilename("tfupdate")),
				pipeline.WorkspacePath("custom-artifacts", ci.ReportFilename("tfupdate")),
			}
			if !slices.Equal(producedPaths(produces), wantPaths) {
				t.Fatalf("tfupdate-check produced paths = %v, want %v", producedPaths(produces), wantPaths)
			}
			foundUpdateArtifactPath = true
		}
	}

	if !foundUpdateArtifactPath {
		t.Fatal("CollectContributions() did not include dependency-update-check job")
	}
}

func TestCollectContributions_TfupdatePipelineGate(t *testing.T) {
	appCtx, plugins := loadPluginContractConfig(t, `structure:
  pattern: "{service}/{environment}/{region}/{module}"
extensions:
  tfupdate:
    enabled: true
    pipeline: false
`)

	contributions, err := plugins.CollectContributions(appCtx)
	if err != nil {
		t.Fatalf("CollectContributions() error = %v", err)
	}
	for _, contrib := range contributions.Contributions() {
		for _, job := range contrib.Jobs() {
			if job.Name() == "tfupdate-check" {
				t.Fatal("CollectContributions() included tfupdate-check when pipeline=false")
			}
		}
	}
}

func producedPaths(resources []pipeline.ResourceSpec) []string {
	paths := make([]string, 0, len(resources))
	for _, resource := range resources {
		paths = append(paths, resource.Path)
	}
	return paths
}

func loadPluginContractConfig(t *testing.T, rawConfig string) (*plugin.AppContext, *registry.Registry) {
	t.Helper()

	clearCIEnv(t)
	plugins := registry.New()

	dir := t.TempDir()
	configPath := filepath.Join(dir, ".terraci.yaml")
	if err := os.WriteFile(configPath, []byte(rawConfig), 0o600); err != nil {
		t.Fatalf("failed to write config fixture: %v", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config fixture: %v", err)
	}

	if err := plugins.DecodeConfig(cfg); err != nil {
		t.Fatalf("failed to decode plugin config: %v", err)
	}

	serviceDir := filepath.Join(dir, cfg.ServiceDir)
	appCtx := plugin.NewAppContext(plugin.AppContextOptions{
		Config:     cfg,
		WorkDir:    dir,
		ServiceDir: serviceDir,
		Version:    "test",
		Resolvers: plugin.NewResolverSet(plugin.ResolverSetOptions{
			CI:             plugins,
			ChangeDetector: plugins,
			KVCache:        plugins,
			BlobStore:      plugins,
		}),
	})
	return appCtx, plugins
}
