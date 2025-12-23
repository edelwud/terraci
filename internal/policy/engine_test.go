package policy

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestOPAVersion(t *testing.T) {
	version := OPAVersion()
	if version == "" {
		t.Error("OPAVersion() returned empty string")
	}
}

func TestNewEngine(t *testing.T) {
	policyDirs := []string{"/policies"}
	namespaces := []string{"terraform"}

	engine := NewEngine(policyDirs, namespaces)

	if engine == nil {
		t.Fatal("NewEngine() returned nil")
	}
	if len(engine.policyDirs) != 1 {
		t.Errorf("policyDirs = %v, want 1 element", engine.policyDirs)
	}
	if len(engine.namespaces) != 1 {
		t.Errorf("namespaces = %v, want 1 element", engine.namespaces)
	}
}

func TestEngine_Evaluate_NoPolicies(t *testing.T) {
	// Create a temporary plan.json
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.json")
	err := os.WriteFile(planPath, []byte(`{"format_version": "1.0"}`), 0o644)
	if err != nil {
		t.Fatalf("failed to write plan.json: %v", err)
	}

	// Engine with non-existent policy dir
	engine := NewEngine([]string{filepath.Join(tmpDir, "nonexistent")}, []string{"terraform"})

	result, err := engine.Evaluate(context.Background(), planPath)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}

	if result == nil {
		t.Fatal("Evaluate() returned nil result")
	}
	if len(result.Failures) != 0 {
		t.Errorf("expected no failures, got %d", len(result.Failures))
	}
}

func TestEngine_Evaluate_WithPolicy(t *testing.T) {
	tmpDir := t.TempDir()

	// Create plan.json with a resource
	planPath := filepath.Join(tmpDir, "plan.json")
	planJSON := `{
		"format_version": "1.0",
		"resource_changes": [
			{
				"type": "aws_s3_bucket",
				"name": "test",
				"change": {
					"actions": ["create"]
				}
			}
		]
	}`
	if err := os.WriteFile(planPath, []byte(planJSON), 0o644); err != nil {
		t.Fatalf("failed to write plan.json: %v", err)
	}

	// Create policy directory and a simple deny policy
	policyDir := filepath.Join(tmpDir, "policies")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("failed to create policy dir: %v", err)
	}

	policy := `package terraform

deny contains msg if {
	input.resource_changes[_].type == "aws_s3_bucket"
	msg := "S3 buckets are not allowed"
}`
	if err := os.WriteFile(filepath.Join(policyDir, "s3.rego"), []byte(policy), 0o644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	engine := NewEngine([]string{policyDir}, []string{"terraform"})

	result, err := engine.Evaluate(context.Background(), planPath)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}

	if len(result.Failures) != 1 {
		t.Errorf("expected 1 failure, got %d", len(result.Failures))
	}
	if len(result.Failures) > 0 && result.Failures[0].Message != "S3 buckets are not allowed" {
		t.Errorf("unexpected failure message: %s", result.Failures[0].Message)
	}
}

func TestEngine_Evaluate_WithWarn(t *testing.T) {
	tmpDir := t.TempDir()

	// Create plan.json
	planPath := filepath.Join(tmpDir, "plan.json")
	planJSON := `{
		"format_version": "1.0",
		"resource_changes": [
			{
				"type": "aws_instance",
				"name": "test",
				"change": {
					"actions": ["create"]
				}
			}
		]
	}`
	if err := os.WriteFile(planPath, []byte(planJSON), 0o644); err != nil {
		t.Fatalf("failed to write plan.json: %v", err)
	}

	// Create policy with warn rule
	policyDir := filepath.Join(tmpDir, "policies")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("failed to create policy dir: %v", err)
	}

	policy := `package terraform

warn contains msg if {
	input.resource_changes[_].type == "aws_instance"
	msg := "Consider using auto scaling groups"
}`
	if err := os.WriteFile(filepath.Join(policyDir, "instance.rego"), []byte(policy), 0o644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	engine := NewEngine([]string{policyDir}, []string{"terraform"})

	result, err := engine.Evaluate(context.Background(), planPath)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}

	if len(result.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(result.Warnings))
	}
	if len(result.Warnings) > 0 && result.Warnings[0].Message != "Consider using auto scaling groups" {
		t.Errorf("unexpected warning message: %s", result.Warnings[0].Message)
	}
}

func TestEngine_Evaluate_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan.json")
	err := os.WriteFile(planPath, []byte(`{invalid json}`), 0o644)
	if err != nil {
		t.Fatalf("failed to write plan.json: %v", err)
	}

	engine := NewEngine([]string{tmpDir}, []string{"terraform"})

	_, err = engine.Evaluate(context.Background(), planPath)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestEngine_Evaluate_FileNotFound(t *testing.T) {
	engine := NewEngine([]string{"/tmp"}, []string{"terraform"})

	_, err := engine.Evaluate(context.Background(), "/nonexistent/plan.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestEngine_collectRegoFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create policy files
	policyDir := filepath.Join(tmpDir, "policies")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("failed to create policy dir: %v", err)
	}

	// Create some rego files
	files := []string{"policy1.rego", "policy2.rego", "policy_test.rego"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(policyDir, f), []byte("package test"), 0o644); err != nil {
			t.Fatalf("failed to write %s: %v", f, err)
		}
	}

	// Create a non-rego file
	if err := os.WriteFile(filepath.Join(policyDir, "readme.md"), []byte("# Readme"), 0o644); err != nil {
		t.Fatalf("failed to write readme: %v", err)
	}

	engine := NewEngine([]string{policyDir}, []string{"test"})
	regoFiles, err := engine.collectRegoFiles()
	if err != nil {
		t.Fatalf("collectRegoFiles() error = %v", err)
	}

	// Should find 2 rego files (excluding _test.rego)
	if len(regoFiles) != 2 {
		t.Errorf("expected 2 rego files, got %d: %v", len(regoFiles), regoFiles)
	}
}
