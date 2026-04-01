package model_test

import (
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
		{"valid defaults", model.CostConfig{Enabled: true}, false},
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
		kind model.CostErrorKind
		want bool
	}{
		{model.CostErrorNone, false},
		{model.CostErrorUsageBased, false},
		{model.CostErrorNoHandler, true},
		{model.CostErrorLookupFailed, true},
		{model.CostErrorAPIFailure, true},
		{model.CostErrorNoPrice, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			t.Parallel()
			rc := model.ResourceCost{ErrorKind: tt.kind}
			if rc.IsUnsupported() != tt.want {
				t.Errorf("IsUnsupported(%q) = %v, want %v", tt.kind, rc.IsUnsupported(), tt.want)
			}
		})
	}
}

func TestSubmoduleCost_TotalCost(t *testing.T) {
	t.Parallel()

	s := model.SubmoduleCost{
		MonthlyCost: 10,
		Children: []model.SubmoduleCost{
			{MonthlyCost: 20, Children: []model.SubmoduleCost{{MonthlyCost: 30}}},
		},
	}
	if s.TotalCost() != 60 {
		t.Errorf("TotalCost() = %v, want 60", s.TotalCost())
	}

	leaf := model.SubmoduleCost{MonthlyCost: 42}
	if leaf.TotalCost() != 42 {
		t.Errorf("TotalCost() leaf = %v, want 42", leaf.TotalCost())
	}
}
