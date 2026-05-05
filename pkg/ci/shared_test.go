package ci

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestCommentEnabled(t *testing.T) {
	t.Run("nil config defaults true", func(t *testing.T) {
		if !CommentEnabled(nil) {
			t.Fatal("expected nil config to default to true")
		}
	})

	t.Run("nil enabled defaults true", func(t *testing.T) {
		if !CommentEnabled(&MRCommentConfig{}) {
			t.Fatal("expected nil enabled to default to true")
		}
	})

	t.Run("explicit true", func(t *testing.T) {
		enabled := true
		if !CommentEnabled(&MRCommentConfig{Enabled: &enabled}) {
			t.Fatal("expected explicit true")
		}
	})

	t.Run("explicit false", func(t *testing.T) {
		enabled := false
		if CommentEnabled(&MRCommentConfig{Enabled: &enabled}) {
			t.Fatal("expected explicit false")
		}
	})
}

func TestImage_YAMLRoundTrip_Shorthand(t *testing.T) {
	// Shorthand input should remain shorthand on output (no entrypoint).
	const yamlIn = "hashicorp/terraform:1.6\n"

	var img Image
	if err := yaml.Unmarshal([]byte(yamlIn), &img); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if img.Name != "hashicorp/terraform:1.6" {
		t.Errorf("Name = %q, want hashicorp/terraform:1.6", img.Name)
	}

	out, err := yaml.Marshal(img)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.TrimSpace(string(out)) != "hashicorp/terraform:1.6" {
		t.Errorf("round-trip mismatch:\ngot:  %q\nwant: %q", string(out), yamlIn)
	}
}

func TestImage_YAMLRoundTrip_FullForm(t *testing.T) {
	// When entrypoint is set, output expands to the full mapping form.
	img := Image{
		Name:       "ghcr.io/opentofu/opentofu:1.9-minimal",
		Entrypoint: []string{""},
	}
	out, err := yaml.Marshal(img)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	got := string(out)
	if !strings.Contains(got, "name: ghcr.io/opentofu/opentofu:1.9-minimal") {
		t.Errorf("expected full-form YAML to include 'name:', got: %s", got)
	}
	if !strings.Contains(got, "entrypoint:") {
		t.Errorf("expected full-form YAML to include 'entrypoint:', got: %s", got)
	}
}

func TestHasCommentMarker(t *testing.T) {
	if !HasCommentMarker("before " + CommentMarker + " after") {
		t.Fatal("expected marker to be detected")
	}
	if HasCommentMarker("plain body") {
		t.Fatal("expected plain body to not match marker")
	}
}
