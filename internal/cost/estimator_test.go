package cost

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/edelwud/terraci/internal/terraform/plan"
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

func TestGetResourceAttrs(t *testing.T) {
	tests := []struct {
		name     string
		rc       plan.ResourceChange
		wantKeys []string
	}{
		{
			name: "extracts new values from attributes",
			rc: plan.ResourceChange{
				Attributes: []plan.AttrDiff{
					{Path: "instance_type", NewValue: "t3.micro", OldValue: ""},
					{Path: "ami", NewValue: "ami-12345", OldValue: ""},
				},
			},
			wantKeys: []string{"instance_type", "ami"},
		},
		{
			name: "falls back to old value when new is empty",
			rc: plan.ResourceChange{
				Attributes: []plan.AttrDiff{
					{Path: "instance_type", OldValue: "t2.micro", NewValue: ""},
				},
			},
			wantKeys: []string{"instance_type"},
		},
		{
			name: "skips known after apply",
			rc: plan.ResourceChange{
				Attributes: []plan.AttrDiff{
					{Path: "id", NewValue: "(known after apply)", OldValue: ""},
					{Path: "ami", NewValue: "ami-12345", OldValue: ""},
				},
			},
			wantKeys: []string{"ami"},
		},
		{
			name:     "empty attributes returns empty map",
			rc:       plan.ResourceChange{},
			wantKeys: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getResourceAttrs(tt.rc)
			for _, key := range tt.wantKeys {
				if _, ok := got[key]; !ok {
					t.Errorf("getResourceAttrs() missing key %q", key)
				}
			}
			// Verify no extra keys for empty case
			if tt.wantKeys == nil && len(got) != 0 {
				t.Errorf("expected empty map, got %v", got)
			}
		})
	}
}

func TestParseStateResources(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
		wantKey string
	}{
		{
			name: "basic state with one instance",
			input: `{
				"resources": [
					{
						"type": "aws_instance",
						"name": "web",
						"instances": [
							{
								"attributes": {
									"instance_type": "t3.micro",
									"ami": "ami-12345"
								}
							}
						]
					}
				]
			}`,
			wantLen: 1,
			wantKey: "aws_instance.web",
		},
		{
			name: "state with module prefix",
			input: `{
				"resources": [
					{
						"type": "aws_vpc",
						"name": "main",
						"module": "module.networking",
						"instances": [
							{
								"attributes": {"cidr_block": "10.0.0.0/16"}
							}
						]
					}
				]
			}`,
			wantLen: 1,
			wantKey: "module.networking.aws_vpc.main",
		},
		{
			name: "state with string index key",
			input: `{
				"resources": [
					{
						"type": "aws_subnet",
						"name": "private",
						"instances": [
							{
								"index_key": "us-east-1a",
								"attributes": {"cidr_block": "10.0.1.0/24"}
							}
						]
					}
				]
			}`,
			wantLen: 1,
			wantKey: `aws_subnet.private["us-east-1a"]`,
		},
		{
			name: "state with numeric index key",
			input: `{
				"resources": [
					{
						"type": "aws_subnet",
						"name": "private",
						"instances": [
							{
								"index_key": 0,
								"attributes": {"cidr_block": "10.0.1.0/24"}
							}
						]
					}
				]
			}`,
			wantLen: 1,
			wantKey: "aws_subnet.private[0]",
		},
		{
			name:    "invalid JSON returns nil",
			input:   "not json",
			wantLen: 0,
		},
		{
			name:    "empty resources",
			input:   `{"resources": []}`,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseStateResources([]byte(tt.input))
			if tt.wantLen == 0 {
				if len(got) != 0 {
					t.Errorf("expected nil or empty map, got %v", got)
				}
				return
			}
			if len(got) != tt.wantLen {
				t.Errorf("parseStateResources() returned %d resources, want %d", len(got), tt.wantLen)
				return
			}
			if tt.wantKey != "" {
				if _, ok := got[tt.wantKey]; !ok {
					t.Errorf("parseStateResources() missing key %q, got keys: %v", tt.wantKey, keysOf(got))
				}
			}
		})
	}
}

func keysOf(m map[string]map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestBuildResourceAddress(t *testing.T) {
	tests := []struct {
		module       string
		resourceType string
		name         string
		indexKey     any
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
