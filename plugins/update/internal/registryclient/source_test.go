package registryclient

import "testing"

func TestParseModuleSource(t *testing.T) {
	ns, name, provider, err := ParseModuleSource("terraform-aws-modules/vpc/aws")
	if err != nil {
		t.Fatal(err)
	}
	if ns != "terraform-aws-modules" || name != "vpc" || provider != "aws" {
		t.Errorf("got (%s, %s, %s), want (terraform-aws-modules, vpc, aws)", ns, name, provider)
	}

	_, _, _, err = ParseModuleSource("invalid")
	if err == nil {
		t.Error("expected error for invalid source")
	}
}

func TestParseProviderSource(t *testing.T) {
	ns, typeName, err := ParseProviderSource("hashicorp/aws")
	if err != nil {
		t.Fatal(err)
	}
	if ns != "hashicorp" || typeName != "aws" {
		t.Errorf("got (%s, %s), want (hashicorp, aws)", ns, typeName)
	}

	_, _, err = ParseProviderSource("invalid/format/extra")
	if err == nil {
		t.Error("expected error for invalid source")
	}
}

func TestIsRegistrySource(t *testing.T) {
	tests := []struct {
		source string
		want   bool
	}{
		{"terraform-aws-modules/vpc/aws", true},
		{"./modules/vpc", false},
		{"../modules/vpc", false},
		{"git::https://example.com/vpc.git", false},
		{"github.com/org/repo", true},
		{"only/two", false},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := IsRegistrySource(tt.source); got != tt.want {
				t.Errorf("IsRegistrySource(%q) = %v, want %v", tt.source, got, tt.want)
			}
		})
	}
}
