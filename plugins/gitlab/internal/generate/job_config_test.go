package generate

import (
	"testing"

	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
)

func TestConvertSecrets(t *testing.T) {
	secrets := map[string]configpkg.CfgSecret{
		"AWS_SECRET_KEY": {
			Vault: &configpkg.CfgVaultSecret{
				Shorthand: "secret/data/aws/key@production",
			},
			File: true,
		},
	}

	result := convertSecrets(secrets)
	got := result["AWS_SECRET_KEY"]
	if got == nil {
		t.Fatal("expected converted secret")
	}
	if got.VaultPath != "secret/data/aws/key@production" {
		t.Fatalf("VaultPath = %q", got.VaultPath)
	}
	if !got.File {
		t.Fatal("expected File=true")
	}
}
