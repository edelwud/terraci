package terraformrun

import (
	"testing"

	"github.com/edelwud/terraci/pkg/config"
)

func TestNewProfileDefaults(t *testing.T) {
	profile, err := NewProfile(ProfileOptions{})
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}
	if profile.Binary() != BinaryTerraform {
		t.Fatalf("Binary() = %q, want %q", profile.Binary(), BinaryTerraform)
	}
	if !profile.InitEnabled() {
		t.Fatal("InitEnabled() = false, want true")
	}
	if profile.Parallelism() != DefaultParallelism {
		t.Fatalf("Parallelism() = %d, want %d", profile.Parallelism(), DefaultParallelism)
	}
}

func TestNewProfileValidatesBinary(t *testing.T) {
	if _, err := NewProfile(ProfileOptions{Binary: "bad"}); err == nil {
		t.Fatal("NewProfile() error = nil, want invalid binary error")
	}
}

func TestNewProfileEnvIsDefensive(t *testing.T) {
	env := map[string]string{"TF_VAR": "one"}
	profile, err := NewProfile(ProfileOptions{Env: env})
	if err != nil {
		t.Fatalf("NewProfile() error = %v", err)
	}
	env["TF_VAR"] = "mutated"
	got := profile.Env()
	got["TF_VAR"] = "changed"
	if profile.Env()["TF_VAR"] != "one" {
		t.Fatalf("Env() leaked mutation: %#v", profile.Env())
	}
}

func TestProfileFromConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Execution.Binary = config.ExecutionBinaryTofu
	cfg.Execution.InitEnabled = false
	cfg.Execution.Parallelism = 8
	cfg.Execution.Env = map[string]string{"TF_LOG": "WARN"}

	profile, err := ProfileFromConfig(cfg.Snapshot())
	if err != nil {
		t.Fatalf("ProfileFromConfig() error = %v", err)
	}
	if profile.Binary() != BinaryTofu {
		t.Fatalf("Binary() = %q, want %q", profile.Binary(), BinaryTofu)
	}
	if profile.InitEnabled() {
		t.Fatal("InitEnabled() = true, want false")
	}
	if profile.Parallelism() != 8 {
		t.Fatalf("Parallelism() = %d, want 8", profile.Parallelism())
	}
	if profile.Env()["TF_LOG"] != "WARN" {
		t.Fatalf("Env() = %#v, want TF_LOG", profile.Env())
	}
}
