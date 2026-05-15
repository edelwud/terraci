package ci

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

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

func TestManagedLabelsMetadataRoundTrip(t *testing.T) {
	body := CommentMarker + "\n\n## Terraform Plan"
	withMetadata := EmbedManagedLabels(body, []string{" beta ", "alpha", "alpha"})

	if got := ExtractManagedLabels(withMetadata); strings.Join(got, ",") != "alpha,beta" {
		t.Fatalf("ExtractManagedLabels = %v, want [alpha beta]", got)
	}
	if strings.Count(withMetadata, managedLabelsPrefix) != 1 {
		t.Fatalf("expected one metadata comment, got body:\n%s", withMetadata)
	}

	updated := EmbedManagedLabels(withMetadata, []string{"gamma"})
	if got := ExtractManagedLabels(updated); strings.Join(got, ",") != "gamma" {
		t.Fatalf("ExtractManagedLabels after update = %v, want [gamma]", got)
	}
	if strings.Count(updated, managedLabelsPrefix) != 1 {
		t.Fatalf("expected metadata replacement, got body:\n%s", updated)
	}
}

func TestDiffManagedLabels(t *testing.T) {
	add, remove := DiffManagedLabels(
		[]string{"stale", "keep", " dup ", "dup"},
		[]string{"keep", "new"},
	)

	if strings.Join(add, ",") != "new" {
		t.Fatalf("add = %v, want [new]", add)
	}
	if strings.Join(remove, ",") != "dup,stale" {
		t.Fatalf("remove = %v, want [dup stale]", remove)
	}
}
