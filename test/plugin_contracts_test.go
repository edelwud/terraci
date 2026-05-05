package test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
)

func TestBuiltInPluginContractMatrix(t *testing.T) {
	expected := map[string]struct {
		configLoader bool
		command      bool
		preflight    bool
		runtime      bool
		pipeline     bool
	}{
		"cost": {
			configLoader: true,
			command:      true,
			preflight:    true,
			runtime:      true,
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
			runtime:      true,
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
			runtime:      true,
			pipeline:     true,
		},
	}

	for _, p := range registry.New().All() {
		want, ok := expected[p.Name()]
		if !ok {
			t.Fatalf("unexpected plugin %q in registry", p.Name())
		}

		_, hasConfigLoader := p.(plugin.ConfigLoader)
		_, hasCommandProvider := p.(plugin.CommandProvider)
		_, hasPreflight := p.(plugin.Preflightable)
		_, hasRuntime := p.(plugin.RuntimeProvider)
		_, hasPipeline := p.(plugin.PipelineContributor)

		if hasConfigLoader != want.configLoader {
			t.Errorf("%s ConfigLoader = %v, want %v", p.Name(), hasConfigLoader, want.configLoader)
		}
		if hasCommandProvider != want.command {
			t.Errorf("%s CommandProvider = %v, want %v", p.Name(), hasCommandProvider, want.command)
		}
		if hasPreflight != want.preflight {
			t.Errorf("%s Preflightable = %v, want %v", p.Name(), hasPreflight, want.preflight)
		}
		if hasRuntime != want.runtime {
			t.Errorf("%s RuntimeProvider = %v, want %v", p.Name(), hasRuntime, want.runtime)
		}
		if hasPipeline != want.pipeline {
			t.Errorf("%s PipelineContributor = %v, want %v", p.Name(), hasPipeline, want.pipeline)
		}
	}
}

func TestPreflightsForStartup_UsesEnabledPlugins(t *testing.T) {
	appCtx := loadPluginContractConfig(t, `service_dir: .terraci
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
      - path: terraform
  summary: {}
  tfupdate:
    enabled: true
`)

	plugins := appCtx.Resolver().(*registry.Registry)
	preflightables := plugins.PreflightsForStartup()
	got := make([]string, 0, len(preflightables))
	for _, p := range preflightables {
		if err := p.Preflight(context.Background(), appCtx); err != nil && p.Name() != "git" {
			t.Fatalf("Preflight(%s) error = %v", p.Name(), err)
		}
		got = append(got, p.Name())
	}
	slices.Sort(got)

	want := []string{"cost", "git", "gitlab", "policy", "tfupdate"}
	if !slices.Equal(got, want) {
		t.Fatalf("PreflightsForStartup() = %v, want %v", got, want)
	}
}

func TestRuntimeProviders_CreateRuntimeWithoutPreflight(t *testing.T) {
	appCtx := loadPluginContractConfig(t, `service_dir: .terraci
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
      - path: terraform
  tfupdate:
    enabled: true
    policy:
      bump: minor
`)

	expectedRuntimeProviders := []string{"cost", "policy", "tfupdate"}
	got := make([]string, 0, len(expectedRuntimeProviders))
	plugins := appCtx.Resolver().(*registry.Registry)
	for _, p := range registry.ByCapabilityFrom[plugin.RuntimeProvider](plugins) {
		rawRuntime, err := p.Runtime(context.Background(), appCtx)
		if err != nil {
			t.Fatalf("Runtime(%s) error = %v", p.Name(), err)
		}
		if rawRuntime == nil {
			t.Fatalf("Runtime(%s) returned nil runtime", p.Name())
		}
		got = append(got, p.Name())
	}
	slices.Sort(got)
	if !slices.Equal(got, expectedRuntimeProviders) {
		t.Fatalf("Runtime providers = %v, want %v", got, expectedRuntimeProviders)
	}
}

func TestCollectContributions_UsesContextualStateWithoutPreflight(t *testing.T) {
	appCtx := loadPluginContractConfig(t, `service_dir: custom-artifacts
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
      - path: terraform
  summary: {}
  tfupdate:
    enabled: true
    pipeline: true
`)

	plugins := appCtx.Resolver().(*registry.Registry)
	contributions := plugins.CollectContributions(appCtx)
	if len(contributions) != 4 {
		t.Fatalf("CollectContributions() returned %d contributions, want 4", len(contributions))
	}

	foundUpdateArtifactPath := false
	for _, contrib := range contributions {
		for _, job := range contrib.Jobs {
			if job.Name != "tfupdate-check" {
				continue
			}
			if len(job.ArtifactPaths) != 1 {
				t.Fatalf("tfupdate-check artifact paths = %v, want one path", job.ArtifactPaths)
			}
			if job.ArtifactPaths[0] != filepath.Join("custom-artifacts", "tfupdate-results.json") {
				t.Fatalf("dtfupdate-check artifact path = %q, want %q", job.ArtifactPaths[0], filepath.Join("custom-artifacts", "tfupdate-results.json"))
			}
			foundUpdateArtifactPath = true
		}
	}

	if !foundUpdateArtifactPath {
		t.Fatal("CollectContributions() did not include dependency-update-check job")
	}
}

func loadPluginContractConfig(t *testing.T, rawConfig string) *plugin.AppContext {
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

	for _, cl := range registry.ByCapabilityFrom[plugin.ConfigLoader](plugins) {
		if _, exists := cfg.Extensions[cl.ConfigKey()]; !exists {
			continue
		}
		if err := cl.DecodeAndSet(func(target any) error {
			return cfg.Extension(cl.ConfigKey(), target)
		}); err != nil {
			t.Fatalf("failed to decode %s config: %v", cl.ConfigKey(), err)
		}
	}

	serviceDir := filepath.Join(dir, cfg.ServiceDir)
	return plugin.NewAppContext(plugin.AppContextOptions{
		Config:     cfg,
		WorkDir:    dir,
		ServiceDir: serviceDir,
		Version:    "test",
		Resolver:   plugins,
	})
}
