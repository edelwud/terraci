package config

import "testing"

func TestBuild_WithPattern(t *testing.T) {
	t.Parallel()

	cfg, err := Build(BuildOptions{Pattern: "{service}/{environment}/{module}"})
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

func TestBuild_EmptyPattern(t *testing.T) {
	t.Parallel()

	cfg, err := Build(BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// Should use default pattern
	if cfg.Structure.Pattern == "" {
		t.Error("expected default pattern, got empty")
	}
}

func TestBuild_ProviderA(t *testing.T) {
	t.Parallel()

	extensions := mustExtensionSet(t, "provider_a", map[string]any{
		"image": map[string]any{"name": "hashicorp/terraform:1.6"},
		"mr": map[string]any{
			"comment": map[string]any{"enabled": true},
		},
	})
	execution := DefaultConfig().Execution
	execution.Binary = "terraform"
	execution.PlanEnabled = true
	execution.InitEnabled = true

	cfg, err := Build(BuildOptions{
		Pattern:    "{service}/{environment}/{module}",
		Execution:  &execution,
		Extensions: extensions,
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := cfg.Extensions["provider_a"]; !ok {
		t.Fatal("expected provider_a in extensions")
	}
	if _, ok := cfg.Extensions["provider_b"]; ok {
		t.Error("expected provider_b to be absent")
	}

	var providerACfg map[string]any
	if err := cfg.Extension("provider_a", &providerACfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Execution.PlanEnabled != true {
		t.Error("execution.plan_enabled should be true")
	}
	if providerACfg["mr"] == nil {
		t.Error("mr config should be present")
	}
}

func TestBuild_ProviderB(t *testing.T) {
	t.Parallel()

	extensions := mustExtensionSet(t, "provider_b", map[string]any{
		"runs_on": "ubuntu-latest",
		"pr": map[string]any{
			"comment": map[string]any{},
		},
	})
	execution := DefaultConfig().Execution
	execution.Binary = "tofu"
	execution.PlanEnabled = true
	execution.InitEnabled = true

	cfg, err := Build(BuildOptions{Execution: &execution, Extensions: extensions})
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := cfg.Extensions["provider_b"]; !ok {
		t.Fatal("expected provider_b in extensions")
	}
	if _, ok := cfg.Extensions["provider_a"]; ok {
		t.Error("expected provider_a to be absent")
	}

	var providerBCfg map[string]any
	if err := cfg.Extension("provider_b", &providerBCfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Execution.Binary != "tofu" {
		t.Errorf("binary = %v, want tofu", cfg.Execution.Binary)
	}
	if providerBCfg["pr"] == nil {
		t.Error("pr config should be present")
	}
}

func TestBuild_WithFeature(t *testing.T) {
	t.Parallel()

	extensions := mustExtensionSet(t, "feature_a", map[string]any{"enabled": true})
	execution := DefaultConfig().Execution
	execution.Binary = "terraform"

	cfg, err := Build(BuildOptions{Execution: &execution, Extensions: extensions})
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := cfg.Extensions["feature_a"]; !ok {
		t.Error("expected feature_a in extensions")
	}

	var featureCfg map[string]any
	if err := cfg.Extension("feature_a", &featureCfg); err != nil {
		t.Fatal(err)
	}
	if featureCfg["enabled"] != true {
		t.Error("feature_a should be enabled")
	}
}

func TestBuild_InvalidPattern(t *testing.T) {
	t.Parallel()

	_, err := Build(BuildOptions{Pattern: "{service}/{service}"})
	if err == nil {
		t.Fatal("expected validation error for invalid pattern")
	}
}

func TestBuild_InvalidExecution(t *testing.T) {
	t.Parallel()

	execution := DefaultConfig().Execution
	execution.Binary = "terragrunt"
	_, err := Build(BuildOptions{Execution: &execution})
	if err == nil {
		t.Fatal("expected validation error for invalid execution.binary")
	}
}

func TestNewExtensionSet_DuplicateKey(t *testing.T) {
	t.Parallel()

	first := mustExtensionValue(t, "feature", map[string]any{"enabled": true})
	second := mustExtensionValue(t, "feature", map[string]any{"enabled": false})
	if _, err := NewExtensionSet(first, second); err == nil {
		t.Fatal("NewExtensionSet() error = nil, want duplicate key error")
	}
}

func TestNewExtensionValue_InvalidInput(t *testing.T) {
	t.Parallel()

	if _, err := NewExtensionValue("", map[string]any{"enabled": true}); err == nil {
		t.Fatal("NewExtensionValue() error = nil, want key error")
	}
	if _, err := NewExtensionValue("bad.key", map[string]any{"enabled": true}); err == nil {
		t.Fatal("NewExtensionValue() error = nil, want invalid key error")
	}
	if _, err := NewExtensionValue("feature", nil); err == nil {
		t.Fatal("NewExtensionValue() error = nil, want nil config error")
	}
}

func TestExtensionValue_DefensiveNodeCopy(t *testing.T) {
	t.Parallel()

	value := mustExtensionValue(t, "feature", map[string]any{"enabled": true})
	node := value.Node()
	node.Content = nil
	if len(node.Content) != 0 {
		t.Fatal("mutated defensive node copy still has content")
	}

	var decoded map[string]any
	if err := value.Decode(&decoded); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if decoded["enabled"] != true {
		t.Fatalf("decoded = %#v, want enabled true", decoded)
	}
}

func TestExtensionSet_ValuesAreDefensive(t *testing.T) {
	t.Parallel()

	set := mustExtensionSet(t, "feature", map[string]any{"enabled": true})
	values := set.Values()
	if len(values) != 1 {
		t.Fatalf("Values() len = %d, want 1", len(values))
	}
	node := values[0].Node()
	node.Content = nil
	if len(node.Content) != 0 {
		t.Fatal("mutated defensive node copy still has content")
	}

	cfg, err := Build(BuildOptions{Extensions: set})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	var decoded map[string]any
	if err := cfg.Extension("feature", &decoded); err != nil {
		t.Fatalf("Extension() error = %v", err)
	}
	if decoded["enabled"] != true {
		t.Fatalf("decoded = %#v, want enabled true", decoded)
	}
}

func mustExtensionSet(tb testing.TB, key string, value any) ExtensionSet {
	tb.Helper()
	extension := mustExtensionValue(tb, key, value)
	set, err := NewExtensionSet(extension)
	if err != nil {
		tb.Fatalf("NewExtensionSet() error = %v", err)
	}
	return set
}

func mustExtensionValue(tb testing.TB, key string, value any) ExtensionValue {
	tb.Helper()
	extension, err := NewExtensionValue(key, value)
	if err != nil {
		tb.Fatalf("NewExtensionValue() error = %v", err)
	}
	return extension
}
