package config

import "testing"

func TestBuildConfigFromPlugins_WithPattern(t *testing.T) {
	t.Parallel()

	cfg := BuildConfigFromPlugins("{service}/{environment}/{module}", nil)

	if cfg.Structure.Pattern != "{service}/{environment}/{module}" {
		t.Errorf("pattern = %q, want {service}/{environment}/{module}", cfg.Structure.Pattern)
	}
	if len(cfg.Structure.Segments) != 3 {
		t.Errorf("segments count = %d, want 3", len(cfg.Structure.Segments))
	}
}

func TestBuildConfigFromPlugins_EmptyPattern(t *testing.T) {
	t.Parallel()

	cfg := BuildConfigFromPlugins("", nil)

	// Should use default pattern
	if cfg.Structure.Pattern == "" {
		t.Error("expected default pattern, got empty")
	}
}

func TestBuildConfigFromPlugins_GitLab(t *testing.T) {
	t.Parallel()

	pluginConfigs := map[string]map[string]any{
		"gitlab": {
			"terraform_binary": "terraform",
			"image":            map[string]any{"name": "hashicorp/terraform:1.6"},
			"plan_enabled":     true,
			"auto_approve":     false,
			"init_enabled":     true,
			"mr": map[string]any{
				"comment": map[string]any{"enabled": true},
			},
		},
	}

	cfg := BuildConfigFromPlugins("{service}/{environment}/{module}", pluginConfigs)

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
	if glCfg["plan_enabled"] != true {
		t.Error("plan_enabled should be true")
	}
	if glCfg["mr"] == nil {
		t.Error("mr config should be present")
	}
}

func TestBuildConfigFromPlugins_GitHub(t *testing.T) {
	t.Parallel()

	pluginConfigs := map[string]map[string]any{
		"github": {
			"terraform_binary": "tofu",
			"runs_on":          "ubuntu-latest",
			"plan_enabled":     true,
			"auto_approve":     true,
			"init_enabled":     true,
			"pr": map[string]any{
				"comment": map[string]any{},
			},
		},
	}

	cfg := BuildConfigFromPlugins("", pluginConfigs)

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
	if ghCfg["terraform_binary"] != "tofu" {
		t.Errorf("binary = %v, want tofu", ghCfg["terraform_binary"])
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
		"gitlab": {
			"terraform_binary": "terraform",
		},
		"cost": {
			"enabled": true,
		},
	}

	cfg := BuildConfigFromPlugins("", pluginConfigs)

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

func TestSetPluginValue(t *testing.T) {
	t.Parallel()

	t.Run("sets value on existing plugin", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultConfig()
		setPluginNode(cfg, "gitlab", map[string]any{})
		SetPluginValue(cfg, "gitlab", "plan_only", true)

		var m map[string]any
		if err := cfg.PluginConfig("gitlab", &m); err != nil {
			t.Fatal(err)
		}
		if m["plan_only"] != true {
			t.Error("expected plan_only true")
		}
	})

	t.Run("sets value on empty config", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultConfig()
		SetPluginValue(cfg, "github", "auto_approve", false)

		var m map[string]any
		if err := cfg.PluginConfig("github", &m); err != nil {
			t.Fatal(err)
		}
		if m["auto_approve"] != false {
			t.Error("expected auto_approve false")
		}
	})
}
