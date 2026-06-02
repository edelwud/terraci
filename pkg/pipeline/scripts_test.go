package pipeline

import (
	"testing"

	"github.com/edelwud/terraci/pkg/terraformrun"
)

func TestNewTerraformJobConfigFromProfile(t *testing.T) {
	t.Parallel()

	initEnabled := false
	env := map[string]string{"CUSTOM": "value"}
	profile, err := terraformrun.NewProfile(terraformrun.ProfileOptions{
		Binary:      "tofu",
		InitEnabled: &initEnabled,
		Env:         env,
	})
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}

	config, err := NewTerraformJobConfigFromProfile(profile)
	if err != nil {
		t.Fatalf("NewTerraformJobConfigFromProfile() error = %v", err)
	}

	op := config.NewApplyOperation("platform/stage/eu-central-1/vpc", false).Terraform()
	if op.Binary() != "tofu" {
		t.Fatalf("Binary() = %q, want tofu", op.Binary())
	}
	if op.InitEnabled() {
		t.Fatal("InitEnabled() = true, want false")
	}

	env["CUSTOM"] = "mutated"
	gotEnv := config.TerraformEnv()
	if gotEnv["CUSTOM"] != "value" {
		t.Fatalf("TerraformEnv()[CUSTOM] = %q, want value", gotEnv["CUSTOM"])
	}
	gotEnv["CUSTOM"] = "mutated"
	if fresh := config.TerraformEnv(); fresh["CUSTOM"] != "value" {
		t.Fatalf("fresh TerraformEnv()[CUSTOM] = %q, want value", fresh["CUSTOM"])
	}
}
