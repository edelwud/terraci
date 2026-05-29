package domain

import "testing"

func TestImageConfigMarshalYAML(t *testing.T) {
	out, err := ImageConfig{Name: "hashicorp/terraform:1.6"}.MarshalYAML()
	if err != nil {
		t.Fatalf("MarshalYAML() error = %v", err)
	}

	got, ok := out.(string)
	if !ok {
		t.Fatalf("MarshalYAML() type = %T, want string", out)
	}
	if got != "hashicorp/terraform:1.6" {
		t.Fatalf("MarshalYAML() = %q", got)
	}
}
