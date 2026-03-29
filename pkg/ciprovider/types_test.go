package ciprovider

import (
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestImage_UnmarshalYAML_StringShorthand(t *testing.T) {
	input := `"hashicorp/terraform:1.6"`
	var img Image
	if err := yaml.Unmarshal([]byte(input), &img); err != nil {
		t.Fatal(err)
	}
	if img.Name != "hashicorp/terraform:1.6" {
		t.Errorf("Name = %q, want hashicorp/terraform:1.6", img.Name)
	}
	if len(img.Entrypoint) != 0 {
		t.Errorf("Entrypoint = %v, want empty", img.Entrypoint)
	}
}

func TestImage_UnmarshalYAML_ObjectSyntax(t *testing.T) {
	input := `
name: "my-image:latest"
entrypoint: ["/bin/sh", "-c"]
`
	var img Image
	if err := yaml.Unmarshal([]byte(input), &img); err != nil {
		t.Fatal(err)
	}
	if img.Name != "my-image:latest" {
		t.Errorf("Name = %q, want my-image:latest", img.Name)
	}
	if len(img.Entrypoint) != 2 || img.Entrypoint[0] != "/bin/sh" {
		t.Errorf("Entrypoint = %v, want [/bin/sh -c]", img.Entrypoint)
	}
}

func TestImage_String(t *testing.T) {
	img := &Image{Name: "test:1.0"}
	if img.String() != "test:1.0" {
		t.Errorf("String() = %q, want test:1.0", img.String())
	}
}

func TestImage_HasEntrypoint(t *testing.T) {
	img := &Image{Name: "test"}
	if img.HasEntrypoint() {
		t.Error("HasEntrypoint() should be false when no entrypoint")
	}

	img.Entrypoint = []string{"/bin/sh"}
	if !img.HasEntrypoint() {
		t.Error("HasEntrypoint() should be true when entrypoint set")
	}
}

func TestImage_MarshalRoundtrip(t *testing.T) {
	original := Image{Name: "test:1.0", Entrypoint: []string{"/bin/sh"}}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}

	var decoded Image
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.Name != original.Name {
		t.Errorf("Name roundtrip: got %q, want %q", decoded.Name, original.Name)
	}
	if len(decoded.Entrypoint) != len(original.Entrypoint) {
		t.Errorf("Entrypoint roundtrip: got %v, want %v", decoded.Entrypoint, original.Entrypoint)
	}
}

func TestMRCommentConfig_Defaults(t *testing.T) {
	input := `{}`
	var cfg MRCommentConfig
	if err := yaml.Unmarshal([]byte(input), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Enabled != nil {
		t.Error("Enabled should be nil by default")
	}
	if cfg.OnChangesOnly {
		t.Error("OnChangesOnly should be false by default")
	}
	if cfg.IncludeDetails {
		t.Error("IncludeDetails should be false by default")
	}
}

func TestMRCommentConfig_FullConfig(t *testing.T) {
	input := `
enabled: true
on_changes_only: true
include_details: true
`
	var cfg MRCommentConfig
	if err := yaml.Unmarshal([]byte(input), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Enabled == nil || !*cfg.Enabled {
		t.Error("Enabled should be true")
	}
	if !cfg.OnChangesOnly {
		t.Error("OnChangesOnly should be true")
	}
	if !cfg.IncludeDetails {
		t.Error("IncludeDetails should be true")
	}
}

func TestMRCommentConfig_DisabledExplicitly(t *testing.T) {
	input := `enabled: false`
	var cfg MRCommentConfig
	if err := yaml.Unmarshal([]byte(input), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Enabled == nil || *cfg.Enabled {
		t.Error("Enabled should be false")
	}
}
