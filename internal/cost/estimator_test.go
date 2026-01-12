package cost

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEstimator_EstimateModule_WithMockPlan(t *testing.T) {
	// Create temp directory with mock plan.json
	tmpDir := t.TempDir()
	modulePath := filepath.Join(tmpDir, "platform", "prod", "eu-central-1", "vpc")
	if err := os.MkdirAll(modulePath, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write minimal plan.json
	planJSON := `{
		"format_version": "1.2",
		"terraform_version": "1.6.0",
		"resource_changes": [
			{
				"address": "aws_instance.web",
				"type": "aws_instance",
				"name": "web",
				"change": {
					"actions": ["create"],
					"before": null,
					"after": {
						"instance_type": "t3.micro",
						"ami": "ami-12345"
					}
				}
			}
		]
	}`

	if err := os.WriteFile(filepath.Join(modulePath, "plan.json"), []byte(planJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	// Create estimator with temp cache dir
	cacheDir := filepath.Join(tmpDir, "cache")
	estimator := NewEstimator(cacheDir, 24*time.Hour)

	// Test that estimator runs without panic (won't have real pricing data)
	ctx := context.Background()
	result, err := estimator.EstimateModule(ctx, modulePath, "eu-central-1")

	// We expect this to work but have unsupported resources (no cache)
	if err != nil {
		t.Logf("Expected error due to no pricing cache: %v", err)
	}

	if result != nil {
		t.Logf("Module cost result: before=$%.2f, after=$%.2f, diff=$%.2f",
			result.BeforeCost, result.AfterCost, result.DiffCost)
	}
}

func TestEstimator_ValidateAndPrefetch(t *testing.T) {
	tmpDir := t.TempDir()
	modulePath := filepath.Join(tmpDir, "test", "module")
	if err := os.MkdirAll(modulePath, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write minimal plan.json with EC2 instance
	planJSON := `{
		"format_version": "1.2",
		"terraform_version": "1.6.0",
		"resource_changes": [
			{
				"address": "aws_instance.web",
				"type": "aws_instance",
				"name": "web",
				"change": {
					"actions": ["create"],
					"before": null,
					"after": {"instance_type": "t3.micro"}
				}
			}
		]
	}`

	if err := os.WriteFile(filepath.Join(modulePath, "plan.json"), []byte(planJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	cacheDir := filepath.Join(tmpDir, "cache")
	estimator := NewEstimator(cacheDir, 24*time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	modulePaths := []string{modulePath}
	regions := map[string]string{modulePath: "us-east-1"}

	// This will fail due to timeout (won't actually download), but shouldn't panic
	err := estimator.ValidateAndPrefetch(ctx, modulePaths, regions)
	if err != nil {
		t.Logf("Expected timeout/error: %v", err)
	}
}

func TestNewEstimator(t *testing.T) {
	estimator := NewEstimator("", 0)
	if estimator == nil {
		t.Fatal("NewEstimator returned nil")
	}
	if estimator.registry == nil {
		t.Error("registry is nil")
	}
	if estimator.cache == nil {
		t.Error("cache is nil")
	}
}

func TestExtractResourceType(t *testing.T) {
	tests := []struct {
		address  string
		expected string
	}{
		{"aws_instance.web", "aws_instance"},
		{"module.vpc.aws_vpc.main", "aws_vpc"},
		{"module.vpc.module.subnets.aws_subnet.private", "aws_subnet"},
		{"google_compute_instance.vm", "google_compute_instance"},
		{"invalid", ""},
	}

	for _, tt := range tests {
		t.Run(tt.address, func(t *testing.T) {
			result := extractResourceType(tt.address)
			if result != tt.expected {
				t.Errorf("extractResourceType(%q) = %q, want %q", tt.address, result, tt.expected)
			}
		})
	}
}

func TestBuildResourceAddress(t *testing.T) {
	tests := []struct {
		module       string
		resourceType string
		name         string
		indexKey     interface{}
		expected     string
	}{
		{"", "aws_instance", "web", nil, "aws_instance.web"},
		{"module.vpc", "aws_vpc", "main", nil, "module.vpc.aws_vpc.main"},
		{"", "aws_instance", "web", "foo", `aws_instance.web["foo"]`},
		{"", "aws_instance", "web", float64(0), "aws_instance.web[0]"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := buildResourceAddress(tt.module, tt.resourceType, tt.name, tt.indexKey)
			if result != tt.expected {
				t.Errorf("buildResourceAddress() = %q, want %q", result, tt.expected)
			}
		})
	}
}
