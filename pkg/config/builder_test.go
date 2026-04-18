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

func TestBuildConfigFromPlugins_GitLab(t *testing.T) {
	t.Parallel()

	pluginConfigs := map[string]map[string]any{
		"gitlab": {
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

	if _, ok := cfg.Plugins["gitlab"]; !ok {
		t.Fatal("expected gitlab in plugins")
	}
	if _, ok := cfg.Plugins["github"]; ok {
		t.Error("expected github to be absent")
	}

	var glCfg map[string]any
	if err := cfg.PluginConfig("gitlab", &glCfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Execution.PlanEnabled != true {
		t.Error("execution.plan_enabled should be true")
	}
	if glCfg["mr"] == nil {
		t.Error("mr config should be present")
	}
}

func TestBuildConfigFromPlugins_GitHub(t *testing.T) {
	t.Parallel()

	pluginConfigs := map[string]map[string]any{
		"github": {
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

	if _, ok := cfg.Plugins["github"]; !ok {
		t.Fatal("expected github in plugins")
	}
	if _, ok := cfg.Plugins["gitlab"]; ok {
		t.Error("expected gitlab to be absent")
	}

	var ghCfg map[string]any
	if err := cfg.PluginConfig("github", &ghCfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Execution.Binary != "tofu" {
		t.Errorf("binary = %v, want tofu", cfg.Execution.Binary)
	}
	if ghCfg["auto_approve"] != true {
		t.Error("auto_approve should be true")
	}
	if ghCfg["pr"] == nil {
		t.Error("pr config should be present")
	}
}

func TestBuildConfigFromPlugins_WithCost(t *testing.T) {
	t.Parallel()

	pluginConfigs := map[string]map[string]any{
		"cost": {
			"enabled": true,
		},
	}

	cfg, err := BuildConfigFromPlugins("", map[string]any{"binary": "terraform"}, pluginConfigs)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := cfg.Plugins["cost"]; !ok {
		t.Error("expected cost in plugins")
	}

	var costCfg map[string]any
	if err := cfg.PluginConfig("cost", &costCfg); err != nil {
		t.Fatal(err)
	}
	if costCfg["enabled"] != true {
		t.Error("cost should be enabled")
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
