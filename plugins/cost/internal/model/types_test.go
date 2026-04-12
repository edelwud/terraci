package model_test

import (
	"encoding/json"
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

func TestFormatCost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		cost     float64
		expected string
	}{
		{0, "$0"},
		{0.005, "<$0.01"},
		{0.01, "$0.01"},
		{0.5, "$0.5"},
		{1, "$1"},
		{1.5, "$1.5"},
		{10.25, "$10.25"},
		{100, "$100"},
		{999.99, "$999.99"},
		{1000, "$1,000"},
		{1234.56, "$1,234.56"},
		{10000, "$10,000"},
		{-5.5, "-$5.5"},
		{-0.005, "<$0.01"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()

			result := model.FormatCost(tt.cost)
			if result != tt.expected {
				t.Errorf("FormatCost(%v) = %q, want %q", tt.cost, result, tt.expected)
			}
		})
	}
}

func TestFormatCostDiff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		diff     float64
		expected string
	}{
		{0, "$0"},
		{5.5, "+$5.5"},
		{100, "+$100"},
		{1000, "+$1,000"},
		{-5.5, "-$5.5"},
		{-100, "-$100"},
		{-1000, "-$1,000"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()

			result := model.FormatCostDiff(tt.diff)
			if result != tt.expected {
				t.Errorf("FormatCostDiff(%v) = %q, want %q", tt.diff, result, tt.expected)
			}
		})
	}
}

func TestCostConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     model.CostConfig
		wantErr bool
	}{
		{"valid defaults", model.CostConfig{}, false},
		{"valid TTL 48h", model.CostConfig{BlobCache: &model.BlobCacheConfig{TTL: "48h"}}, false},
		{"valid TTL 30m", model.CostConfig{BlobCache: &model.BlobCacheConfig{TTL: "30m"}}, false},
		{"empty TTL ok", model.CostConfig{BlobCache: &model.BlobCacheConfig{TTL: ""}}, false},
		{"invalid TTL", model.CostConfig{BlobCache: &model.BlobCacheConfig{TTL: "invalid"}}, true},
		{"bad TTL format", model.CostConfig{BlobCache: &model.BlobCacheConfig{TTL: "24hours"}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResourceCost_IsUnsupported(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status model.ResourceEstimateStatus
		want   bool
	}{
		{model.ResourceEstimateStatusExact, false},
		{model.ResourceEstimateStatusUsageEstimated, false},
		{model.ResourceEstimateStatusUsageUnknown, false},
		{model.ResourceEstimateStatusUnsupported, true},
		{model.ResourceEstimateStatusFailed, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			t.Parallel()
			rc := model.ResourceCost{Status: tt.status}
			if rc.IsUnsupported() != tt.want {
				t.Errorf("IsUnsupported(%q) = %v, want %v", tt.status, rc.IsUnsupported(), tt.want)
			}
		})
	}
}

func TestResourceCost_IsUsageBased(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		rc   model.ResourceCost
		want bool
	}{
		{
			name: "exact priced resource",
			rc:   model.ResourceCost{Status: model.ResourceEstimateStatusExact},
			want: false,
		},
		{
			name: "usage estimated",
			rc: model.ResourceCost{
				Status: model.ResourceEstimateStatusUsageEstimated,
			},
			want: true,
		},
		{
			name: "usage unknown",
			rc: model.ResourceCost{
				Status: model.ResourceEstimateStatusUsageUnknown,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.rc.IsUsageBased(); got != tt.want {
				t.Fatalf("IsUsageBased() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResourceCost_IsFailedAndContributesAfterCost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		status          model.ResourceEstimateStatus
		wantFailed      bool
		wantContributes bool
	}{
		{"exact", model.ResourceEstimateStatusExact, false, true},
		{"usage estimated", model.ResourceEstimateStatusUsageEstimated, false, true},
		{"usage unknown", model.ResourceEstimateStatusUsageUnknown, false, false},
		{"unsupported", model.ResourceEstimateStatusUnsupported, false, false},
		{"failed", model.ResourceEstimateStatusFailed, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rc := model.ResourceCost{Status: tt.status}
			if got := rc.IsFailed(); got != tt.wantFailed {
				t.Fatalf("IsFailed() = %v, want %v", got, tt.wantFailed)
			}
			if got := rc.ContributesAfterCost(); got != tt.wantContributes {
				t.Fatalf("ContributesAfterCost() = %v, want %v", got, tt.wantContributes)
			}
		})
	}
}

func TestResourceCost_JSONShape(t *testing.T) {
	t.Parallel()

	rc := model.ResourceCost{
		Address:      "aws_lambda_function.worker",
		Type:         "aws_lambda_function",
		Region:       "us-east-1",
		MonthlyCost:  12.04,
		PriceSource:  "usage-based",
		Status:       model.ResourceEstimateStatusUsageEstimated,
		StatusDetail: "usage-based estimate derived from provisioned concurrency",
	}

	data, err := json.Marshal(rc)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if got["status"] != string(model.ResourceEstimateStatusUsageEstimated) {
		t.Fatalf("status = %#v, want %q", got["status"], model.ResourceEstimateStatusUsageEstimated)
	}
	if got["status_detail"] != "usage-based estimate derived from provisioned concurrency" {
		t.Fatalf("status_detail = %#v, want detail", got["status_detail"])
	}
	if _, ok := got["error_kind"]; ok {
		t.Fatalf("unexpected legacy field error_kind in JSON: %s", data)
	}
	if _, ok := got["estimate_kind"]; ok {
		t.Fatalf("unexpected legacy field estimate_kind in JSON: %s", data)
	}
}
