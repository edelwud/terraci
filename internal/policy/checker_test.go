package policy

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/edelwud/terraci/pkg/config"
)

func TestNewChecker(t *testing.T) {
	cfg := &config.PolicyConfig{Enabled: true}
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
	cfg := &config.PolicyConfig{Enabled: false}
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

	cfg := &config.PolicyConfig{Enabled: true}
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

	cfg := &config.PolicyConfig{Enabled: true}
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

	cfg := &config.PolicyConfig{Enabled: true}
	checker := NewChecker(cfg, []string{}, tmpDir)

	summary, err := checker.CheckAll(context.Background())
	if err != nil {
		t.Fatalf("CheckAll() error = %v", err)
	}

	if summary.TotalModules != 2 {
		t.Errorf("TotalModules = %v, want %v", summary.TotalModules, 2)
	}
}

func TestChecker_ShouldBlock(t *testing.T) {
	tests := []struct {
		name      string
		onFailure config.PolicyAction
		summary   *Summary
		expected  bool
	}{
		{
			name:      "block on failure with failures",
			onFailure: config.PolicyActionBlock,
			summary:   &Summary{FailedModules: 1},
			expected:  true,
		},
		{
			name:      "block on failure without failures",
			onFailure: config.PolicyActionBlock,
			summary:   &Summary{PassedModules: 1},
			expected:  false,
		},
		{
			name:      "warn on failure with failures",
			onFailure: config.PolicyActionWarn,
			summary:   &Summary{FailedModules: 1},
			expected:  false,
		},
		{
			name:      "ignore on failure with failures",
			onFailure: config.PolicyActionIgnore,
			summary:   &Summary{FailedModules: 1},
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.PolicyConfig{OnFailure: tt.onFailure}
			checker := NewChecker(cfg, []string{}, "/root")

			if got := checker.ShouldBlock(tt.summary); got != tt.expected {
				t.Errorf("ShouldBlock() = %v, want %v", got, tt.expected)
			}
		})
	}
}
