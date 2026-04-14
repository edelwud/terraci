package lockfile

import (
	"testing"

	"github.com/edelwud/terraci/plugins/tfupdate/internal/sourceaddr"
)

func TestParseProviderAddress(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   ProviderAddress
	}{
		{
			name:   "short source",
			source: "hashicorp/aws",
			want: ProviderAddress{
				Hostname:  sourceaddr.DefaultProviderRegistryHostname,
				Namespace: "hashicorp",
				Type:      "aws",
			},
		},
		{
			name:   "fully qualified source",
			source: "registry.opentofu.org/hashicorp/aws",
			want: ProviderAddress{
				Hostname:  "registry.opentofu.org",
				Namespace: "hashicorp",
				Type:      "aws",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseProviderAddress(tt.source)
			if err != nil {
				t.Fatalf("ParseProviderAddress() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ParseProviderAddress() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestProviderAddressLockSource(t *testing.T) {
	address := ProviderAddress{
		Hostname:  "registry.opentofu.org",
		Namespace: "hashicorp",
		Type:      "aws",
	}

	if got := address.LockSource(); got != "registry.opentofu.org/hashicorp/aws" {
		t.Fatalf("LockSource() = %q", got)
	}
}

func TestParseProviderAddress_ShortSourceDefaultsToTerraformRegistry(t *testing.T) {
	address, err := ParseProviderAddress("hashicorp/aws")
	if err != nil {
		t.Fatalf("ParseProviderAddress() error = %v", err)
	}

	if address.Hostname != sourceaddr.DefaultProviderRegistryHostname {
		t.Fatalf("Hostname = %q, want %q", address.Hostname, sourceaddr.DefaultProviderRegistryHostname)
	}
}
