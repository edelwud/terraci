package policyengine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewChecker(t *testing.T) {
	cfg := &Config{Enabled: true}
	policyDirs := []string{"/policies"}
	rootDir := "/root"

	checker := NewChecker(cfg, policyDirs, rootDir)

	if checker == nil {
		t.Fatal("NewChecker() returned nil")
	}
	if checker.config != cfg {
		t.Error("config not set correctly")
	}
	if len(checker.policyDirs) != 1 {
		t.Errorf("policyDirs = %v, want 1 element", checker.policyDirs)
	}
	if checker.rootDir != rootDir {
		t.Errorf("rootDir = %v, want %v", checker.rootDir, rootDir)
	}
}

func TestChecker_CheckModule_Disabled(t *testing.T) {
	cfg := &Config{Enabled: false}
	checker := NewChecker(cfg, []string{}, "/root")

	result, err := checker.CheckModule(context.Background(), "test/module")
	if err != nil {
		t.Fatalf("CheckModule() error = %v", err)
	}

	if result.Module != "test/module" {
		t.Errorf("Module = %v, want %v", result.Module, "test/module")
	}
	if result.Skipped != 1 {
		t.Errorf("Skipped = %v, want %v", result.Skipped, 1)
	}
}

func TestChecker_CheckModule_NoPlanJSON(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{Enabled: true}
	checker := NewChecker(cfg, []string{}, tmpDir)

	_, err := checker.CheckModule(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for missing plan.json")
	}
}

func TestChecker_CheckModule_WithPlan(t *testing.T) {
	tmpDir := t.TempDir()

	// Create module directory with plan.json
	moduleDir := filepath.Join(tmpDir, "test", "module")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("failed to create module dir: %v", err)
	}

	planJSON := `{"format_version": "1.0", "resource_changes": []}`
	if err := os.WriteFile(filepath.Join(moduleDir, "plan.json"), []byte(planJSON), 0o644); err != nil {
		t.Fatalf("failed to write plan.json: %v", err)
	}

	cfg := &Config{Enabled: true}
	checker := NewChecker(cfg, []string{}, tmpDir)

	result, err := checker.CheckModule(context.Background(), "test/module")
	if err != nil {
		t.Fatalf("CheckModule() error = %v", err)
	}

	if result.Module != "test/module" {
		t.Errorf("Module = %v, want %v", result.Module, "test/module")
	}
}

func TestChecker_CheckAll(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two modules with plan.json
	modules := []string{"mod1", "mod2"}
	for _, mod := range modules {
		moduleDir := filepath.Join(tmpDir, mod)
		if err := os.MkdirAll(moduleDir, 0o755); err != nil {
			t.Fatalf("failed to create module dir: %v", err)
		}
		planJSON := `{"format_version": "1.0"}`
		if err := os.WriteFile(filepath.Join(moduleDir, "plan.json"), []byte(planJSON), 0o644); err != nil {
			t.Fatalf("failed to write plan.json: %v", err)
		}
	}

	cfg := &Config{Enabled: true}
	checker := NewChecker(cfg, []string{}, tmpDir)

	summary, err := checker.CheckAll(context.Background())
	if err != nil {
		t.Fatalf("CheckAll() error = %v", err)
	}

	if summary.TotalModules != 2 {
		t.Errorf("TotalModules = %v, want %v", summary.TotalModules, 2)
	}
}

func TestChecker_CheckModule_OverwriteOnFailureWarn(t *testing.T) {
	tmpDir := t.TempDir()

	// Create policy that denies all S3 buckets
	policyDir := filepath.Join(tmpDir, "policies")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "s3.rego"), []byte(`package terraform
import rego.v1
deny contains msg if {
	some r in input.resource_changes
	r.type == "aws_s3_bucket"
	msg := sprintf("S3 bucket '%s' denied", [r.name])
}
warn contains msg if {
	some r in input.resource_changes
	r.type == "aws_s3_bucket"
	msg := sprintf("S3 bucket '%s' has no versioning", [r.name])
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create module with plan.json
	modDir := filepath.Join(tmpDir, "platform", "sandbox", "eu-central-1", "app")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "plan.json"), []byte(`{
		"format_version": "1.0",
		"resource_changes": [{"type": "aws_s3_bucket", "name": "test", "change": {"actions": ["create"]}}]
	}`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		Enabled:   true,
		OnFailure: ActionBlock,
		Overwrites: []Overwrite{
			{Match: "**/sandbox/**", OnFailure: ActionWarn},
		},
	}
	checker := NewChecker(cfg, []string{policyDir}, tmpDir)

	result, err := checker.CheckModule(context.Background(), "platform/sandbox/eu-central-1/app")
	if err != nil {
		t.Fatalf("CheckModule: %v", err)
	}

	// Failures should be reclassified as warnings for sandbox
	if len(result.Failures) != 0 {
		t.Errorf("Failures = %d, want 0 (should be reclassified)", len(result.Failures))
	}
	// Original warn (versioning) + reclassified deny (S3 denied)
	if len(result.Warnings) != 2 {
		t.Errorf("Warnings = %d, want 2 (1 original + 1 reclassified)", len(result.Warnings))
	}
}

func TestChecker_CheckModule_OverwriteOnFailureBlock(t *testing.T) {
	tmpDir := t.TempDir()

	policyDir := filepath.Join(tmpDir, "policies")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "deny.rego"), []byte(`package terraform
import rego.v1
deny contains msg if {
	some r in input.resource_changes
	r.type == "aws_instance"
	msg := "instance denied"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	modDir := filepath.Join(tmpDir, "platform", "prod", "eu-central-1", "app")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "plan.json"), []byte(`{
		"format_version": "1.0",
		"resource_changes": [{"type": "aws_instance", "name": "web", "change": {"actions": ["create"]}}]
	}`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		Enabled:   true,
		OnFailure: ActionWarn, // global: warn
		Overwrites: []Overwrite{
			{Match: "**/prod/**", OnFailure: ActionBlock}, // prod: block
		},
	}
	checker := NewChecker(cfg, []string{policyDir}, tmpDir)

	result, err := checker.CheckModule(context.Background(), "platform/prod/eu-central-1/app")
	if err != nil {
		t.Fatalf("CheckModule: %v", err)
	}

	// Failures should remain as failures (on_failure=block, not warn)
	if len(result.Failures) != 1 {
		t.Errorf("Failures = %d, want 1 (should NOT be reclassified)", len(result.Failures))
	}
	if len(result.Warnings) != 0 {
		t.Errorf("Warnings = %d, want 0", len(result.Warnings))
	}
}

func TestChecker_CheckModule_OverwriteDisabled(t *testing.T) {
	tmpDir := t.TempDir()

	policyDir := filepath.Join(tmpDir, "policies")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "deny.rego"), []byte(`package terraform
import rego.v1
deny contains msg if { msg := "always deny" }
`), 0o644); err != nil {
		t.Fatal(err)
	}

	modDir := filepath.Join(tmpDir, "legacy", "old", "eu-central-1", "db")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "plan.json"), []byte(`{
		"format_version": "1.0",
		"resource_changes": [{"type": "aws_instance", "name": "x", "change": {"actions": ["create"]}}]
	}`), 0o644); err != nil {
		t.Fatal(err)
	}

	disabled := false
	cfg := &Config{
		Enabled:   true,
		OnFailure: ActionBlock,
		Overwrites: []Overwrite{
			{Match: "legacy/**", Enabled: &disabled},
		},
	}
	checker := NewChecker(cfg, []string{policyDir}, tmpDir)

	result, err := checker.CheckModule(context.Background(), "legacy/old/eu-central-1/db")
	if err != nil {
		t.Fatalf("CheckModule: %v", err)
	}

	// Should be skipped entirely — no evaluation
	if result.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", result.Skipped)
	}
	if len(result.Failures) != 0 {
		t.Errorf("Failures = %d, want 0 (disabled module)", len(result.Failures))
	}
}

func TestChecker_CheckModule_OverwriteNamespace(t *testing.T) {
	tmpDir := t.TempDir()

	policyDir := filepath.Join(tmpDir, "policies")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Policy in "terraform" namespace
	if err := os.WriteFile(filepath.Join(policyDir, "tf.rego"), []byte(`package terraform
import rego.v1
deny contains msg if { msg := "terraform deny" }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	// Policy in "compliance" namespace
	if err := os.WriteFile(filepath.Join(policyDir, "comp.rego"), []byte(`package compliance
import rego.v1
deny contains msg if { msg := "compliance deny" }
`), 0o644); err != nil {
		t.Fatal(err)
	}

	modDir := filepath.Join(tmpDir, "platform", "prod", "eu-central-1", "vpc")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "plan.json"), []byte(`{
		"format_version": "1.0",
		"resource_changes": [{"type": "aws_vpc", "name": "main", "change": {"actions": ["create"]}}]
	}`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		Enabled:    true,
		OnFailure:  ActionBlock,
		Namespaces: []string{"terraform"}, // default: only terraform
		Overwrites: []Overwrite{
			{Match: "**/prod/**", Namespaces: []string{"terraform", "compliance"}}, // prod: also compliance
		},
	}
	checker := NewChecker(cfg, []string{policyDir}, tmpDir)

	result, err := checker.CheckModule(context.Background(), "platform/prod/eu-central-1/vpc")
	if err != nil {
		t.Fatalf("CheckModule: %v", err)
	}

	// Should have failures from BOTH namespaces
	if len(result.Failures) != 2 {
		t.Errorf("Failures = %d, want 2 (terraform + compliance)", len(result.Failures))
		for _, f := range result.Failures {
			t.Logf("  failure: %s (ns: %s)", f.Message, f.Namespace)
		}
	}
}

func TestChecker_CheckAll_MixedOverwrites(t *testing.T) {
	tmpDir := t.TempDir()

	policyDir := filepath.Join(tmpDir, "policies")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "deny.rego"), []byte(`package terraform
import rego.v1
deny contains msg if { msg := "always deny" }
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// 3 modules: normal, sandbox (warn), legacy (disabled)
	for _, mod := range []string{"platform/stage/eu-central-1/app", "platform/sandbox/eu-central-1/test", "legacy/old/eu-central-1/db"} {
		dir := filepath.Join(tmpDir, mod)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "plan.json"), []byte(`{"format_version":"1.0","resource_changes":[{"type":"aws_instance","name":"x","change":{"actions":["create"]}}]}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	disabled := false
	cfg := &Config{
		Enabled:   true,
		OnFailure: ActionBlock,
		Overwrites: []Overwrite{
			{Match: "**/sandbox/**", OnFailure: ActionWarn},
			{Match: "legacy/**", Enabled: &disabled},
		},
	}
	checker := NewChecker(cfg, []string{policyDir}, tmpDir)

	summary, err := checker.CheckAll(context.Background())
	if err != nil {
		t.Fatalf("CheckAll: %v", err)
	}

	if summary.TotalModules != 3 {
		t.Errorf("TotalModules = %d, want 3", summary.TotalModules)
	}
	if summary.FailedModules != 1 {
		t.Errorf("FailedModules = %d, want 1 (only stage/app)", summary.FailedModules)
	}
	if summary.WarnedModules != 1 {
		t.Errorf("WarnedModules = %d, want 1 (sandbox reclassified)", summary.WarnedModules)
	}
	if summary.PassedModules != 1 {
		t.Errorf("PassedModules = %d, want 1 (legacy skipped)", summary.PassedModules)
	}
}

func TestChecker_ShouldBlock(t *testing.T) {
	tests := []struct {
		name      string
		onFailure Action
		summary   *Summary
		expected  bool
	}{
		{
			name:      "block on failure with failures",
			onFailure: ActionBlock,
			summary:   &Summary{FailedModules: 1},
			expected:  true,
		},
		{
			name:      "block on failure without failures",
			onFailure: ActionBlock,
			summary:   &Summary{PassedModules: 1},
			expected:  false,
		},
		{
			name:      "warn on failure with failures",
			onFailure: ActionWarn,
			summary:   &Summary{FailedModules: 1},
			expected:  false,
		},
		{
			name:      "ignore on failure with failures",
			onFailure: ActionIgnore,
			summary:   &Summary{FailedModules: 1},
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{OnFailure: tt.onFailure}
			checker := NewChecker(cfg, []string{}, "/root")

			if got := checker.ShouldBlock(tt.summary); got != tt.expected {
				t.Errorf("ShouldBlock() = %v, want %v", got, tt.expected)
			}
		})
	}
}
