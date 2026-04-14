package lockfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLookupLockedProviderAddress(t *testing.T) {
	t.Run("returns_existing_opentofu_address", func(t *testing.T) {
		dir := t.TempDir()
		lockPath := filepath.Join(dir, ".terraform.lock.hcl")
		err := os.WriteFile(lockPath, []byte(`
provider "registry.opentofu.org/hashicorp/aws" {
  version = "5.0.0"
}
`), 0o600)
		if err != nil {
			t.Fatal(err)
		}

		got, status := lookupLockedProviderAddress(lockPath, "hashicorp", "aws")
		if status != lockAddressFound {
			t.Fatal("lookupLockedProviderAddress() did not find existing provider")
		}
		if got.LockSource() != "registry.opentofu.org/hashicorp/aws" {
			t.Fatalf("LockSource() = %q", got.LockSource())
		}
	})

	t.Run("returns_false_when_multiple_matching_hosts_exist", func(t *testing.T) {
		dir := t.TempDir()
		lockPath := filepath.Join(dir, ".terraform.lock.hcl")
		err := os.WriteFile(lockPath, []byte(`
provider "registry.opentofu.org/hashicorp/aws" {
  version = "5.0.0"
}
provider "registry.terraform.io/hashicorp/aws" {
  version = "5.0.0"
}
`), 0o600)
		if err != nil {
			t.Fatal(err)
		}

		_, status := lookupLockedProviderAddress(lockPath, "hashicorp", "aws")
		if status != lockAddressAmbiguous {
			t.Fatalf("lookupLockedProviderAddress() status = %v, want %v", status, lockAddressAmbiguous)
		}
	})
}
