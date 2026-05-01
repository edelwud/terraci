package config

import "testing"

func TestBuildConfigFromPlugins_WithPattern(t *testing.T) {
	t.Parallel()

	cfg, err := BuildConfigFromPlugins("{service}/{environment}/{module}", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Structure.Pattern != "{service}/{environment}/{module}" {
		t.Errorf("pattern = %q, want {service}/{environment}/{module}", cfg.Structure.Pattern)
	}
	if len(cfg.Structure.Segments) != 3 {
		t.Errorf("segments count = %d, want 3", len(cfg.Structure.Segments))
	}
}

func TestBuildConfigFromPlugins_EmptyPattern(t *testing.T) {
	t.Parallel()

	cfg, err := BuildConfigFromPlugins("", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Should use default pattern
	if cfg.Structure.Pattern == "" {
		t.Error("expected default pattern, got empty")
	}
}

func TestBuildConfigFromPlugins_ProviderA(t *testing.T) {
	t.Parallel()

	pluginConfigs := map[string]map[string]any{
		"provider_a": {
			"image":        map[string]any{"name": "hashicorp/terraform:1.6"},
			"auto_approve": false,
			"mr": map[string]any{
				"comment": map[string]any{"enabled": true},
			},
		},
	}

	cfg, err := BuildConfigFromPlugins("{service}/{environment}/{module}", map[string]any{
		"binary":       "terraform",
		"plan_enabled": true,
		"init_enabled": true,
	}, pluginConfigs)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := cfg.Plugins["provider_a"]; !ok {
		t.Fatal("expected provider_a in plugins")
	}
	if _, ok := cfg.Plugins["provider_b"]; ok {
		t.Error("expected provider_b to be absent")
	}

	var providerACfg map[string]any
	if err := cfg.PluginConfig("provider_a", &providerACfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Execution.PlanEnabled != true {
		t.Error("execution.plan_enabled should be true")
	}
	if providerACfg["mr"] == nil {
		t.Error("mr config should be present")
	}
}

func TestBuildConfigFromPlugins_ProviderB(t *testing.T) {
	t.Parallel()

	pluginConfigs := map[string]map[string]any{
		"provider_b": {
			"runs_on":      "ubuntu-latest",
			"auto_approve": true,
			"pr": map[string]any{
				"comment": map[string]any{},
			},
		},
	}

	cfg, err := BuildConfigFromPlugins("", map[string]any{
		"binary":       "tofu",
		"plan_enabled": true,
		"init_enabled": true,
	}, pluginConfigs)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := cfg.Plugins["provider_b"]; !ok {
		t.Fatal("expected provider_b in plugins")
	}
	if _, ok := cfg.Plugins["provider_a"]; ok {
		t.Error("expected provider_a to be absent")
	}

	var providerBCfg map[string]any
	if err := cfg.PluginConfig("provider_b", &providerBCfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Execution.Binary != "tofu" {
		t.Errorf("binary = %v, want tofu", cfg.Execution.Binary)
	}
	if providerBCfg["auto_approve"] != true {
		t.Error("auto_approve should be true")
	}
	if providerBCfg["pr"] == nil {
		t.Error("pr config should be present")
	}
}

func TestBuildConfigFromPlugins_WithFeature(t *testing.T) {
	t.Parallel()

	pluginConfigs := map[string]map[string]any{
		"feature_a": {
			"enabled": true,
		},
	}

	cfg, err := BuildConfigFromPlugins("", map[string]any{"binary": "terraform"}, pluginConfigs)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := cfg.Plugins["feature_a"]; !ok {
		t.Error("expected feature_a in plugins")
	}

	var featureCfg map[string]any
	if err := cfg.PluginConfig("feature_a", &featureCfg); err != nil {
		t.Fatal(err)
	}
	if featureCfg["enabled"] != true {
		t.Error("feature_a should be enabled")
	}
}

func TestBuildConfigFromPlugins_InvalidPattern(t *testing.T) {
	t.Parallel()

	_, err := BuildConfigFromPlugins("{service}/{service}", nil, nil)
	if err == nil {
		t.Fatal("expected validation error for invalid pattern")
	}
}

func TestBuildConfigFromPlugins_InvalidExecution(t *testing.T) {
	t.Parallel()

	_, err := BuildConfigFromPlugins("", map[string]any{"binary": "terragrunt"}, nil)
	if err == nil {
		t.Fatal("expected validation error for invalid execution.binary")
	}
}
