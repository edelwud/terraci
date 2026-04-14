package sourceaddr

import "testing"

func TestParseRegistryModuleSource(t *testing.T) {
	address, err := ParseRegistryModuleSource("terraform-aws-modules/vpc/aws")
	if err != nil {
		t.Fatal(err)
	}
	if address.Hostname != DefaultProviderRegistryHostname || address.Namespace != "terraform-aws-modules" || address.Name != "vpc" || address.Provider != "aws" {
		t.Errorf("got %+v, want terraform-aws-modules/vpc/aws on %s", address, DefaultProviderRegistryHostname)
	}

	_, err = ParseRegistryModuleSource("invalid")
	if err == nil {
		t.Error("expected error for invalid source")
	}

	address, err = ParseRegistryModuleSource("terraform-aws-modules/iam/aws//modules/iam-role-for-service-accounts")
	if err != nil {
		t.Fatal(err)
	}
	if address.Namespace != "terraform-aws-modules" || address.Name != "iam" || address.Provider != "aws" {
		t.Errorf("got %+v, want terraform-aws-modules/iam/aws", address)
	}
	if address.Subdir != "modules/iam-role-for-service-accounts" {
		t.Errorf("Subdir = %q", address.Subdir)
	}
	if address.Source() != "terraform-aws-modules/iam/aws//modules/iam-role-for-service-accounts" {
		t.Errorf("Source() = %q", address.Source())
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

	_, _, err = ParseProviderSource("invalid")
	if err == nil {
		t.Error("expected error for invalid source")
	}
}

func TestIsRegistryModuleSource(t *testing.T) {
	tests := []struct {
		source string
		want   bool
	}{
		{"terraform-aws-modules/vpc/aws", true},
		{"terraform-aws-modules/iam/aws//modules/iam-role-for-service-accounts", true},
		{"./modules/vpc", false},
		{"../modules/vpc", false},
		{"git::https://example.com/vpc.git", false},
		{"github.com/org/repo", true},
		{"only/two", false},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			if got := IsRegistryModuleSource(tt.source); got != tt.want {
				t.Errorf("IsRegistryModuleSource(%q) = %v, want %v", tt.source, got, tt.want)
			}
		})
	}
}
