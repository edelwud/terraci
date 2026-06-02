package initwiz

import "testing"

type contributionConfig struct {
	Enabled bool `yaml:"enabled"`
}

func TestNewInitContribution(t *testing.T) {
	t.Parallel()

	contribution, err := NewInitContribution("feature", contributionConfig{Enabled: true})
	if err != nil {
		t.Fatalf("NewInitContribution() error = %v", err)
	}
	if contribution.PluginKey() != "feature" {
		t.Fatalf("PluginKey() = %q, want feature", contribution.PluginKey())
	}
	if contribution.ExtensionValue().Key().String() != "feature" {
		t.Fatalf("ExtensionValue().Key() = %q, want feature", contribution.ExtensionValue().Key().String())
	}

	var decoded contributionConfig
	if err := contribution.DecodeConfig(&decoded); err != nil {
		t.Fatalf("DecodeConfig() error = %v", err)
	}
	if !decoded.Enabled {
		t.Fatal("DecodeConfig() decoded Enabled=false, want true")
	}
}

func TestNewInitContribution_InvalidPluginKey(t *testing.T) {
	t.Parallel()

	if _, err := NewInitContribution("", contributionConfig{Enabled: true}); err == nil {
		t.Fatal("NewInitContribution() error = nil, want plugin key error")
	}
}

func TestNewInitContribution_NilConfig(t *testing.T) {
	t.Parallel()

	if _, err := NewInitContribution("feature", nil); err == nil {
		t.Fatal("NewInitContribution() error = nil, want nil config error")
	}
}

func TestInitContribution_GettersAreDefensive(t *testing.T) {
	t.Parallel()

	contribution, err := NewInitContribution("feature", contributionConfig{Enabled: true})
	if err != nil {
		t.Fatalf("NewInitContribution() error = %v", err)
	}

	value := contribution.ExtensionValue()
	node := value.Node()
	node.Content = nil
	if len(node.Content) != 0 {
		t.Fatal("mutated defensive node copy still has content")
	}

	var decoded contributionConfig
	if err := contribution.DecodeConfig(&decoded); err != nil {
		t.Fatalf("DecodeConfig() error = %v", err)
	}
	if !decoded.Enabled {
		t.Fatal("mutating returned extension node leaked into contribution")
	}
}
