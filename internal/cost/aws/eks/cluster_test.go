package eks

import (
	"testing"

	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestClusterHandler_Category(t *testing.T) {
	h := &ClusterHandler{}
	if h.Category() != aws.CostCategoryStandard {
		t.Errorf("Category() = %v, want CostCategoryStandard", h.Category())
	}
}

func TestClusterHandler_ServiceCode(t *testing.T) {
	h := &ClusterHandler{}
	if h.ServiceCode() != pricing.ServiceEKS {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceEKS)
	}
}

func TestClusterHandler_BuildLookup(t *testing.T) {
	h := &ClusterHandler{}

	lookup, err := h.BuildLookup("us-east-1", nil)
	if err != nil {
		t.Fatalf("BuildLookup: %v", err)
	}
	if lookup == nil {
		t.Fatal("expected non-nil lookup")
	}
	if lookup.ServiceCode != pricing.ServiceEKS {
		t.Errorf("ServiceCode = %q, want %q", lookup.ServiceCode, pricing.ServiceEKS)
	}
	if lookup.Attributes["usagetype"] != "USE1-AmazonEKS-Hours:perCluster" {
		t.Errorf("usagetype = %q, want USE1-AmazonEKS-Hours:perCluster", lookup.Attributes["usagetype"])
	}
}

func TestClusterHandler_BuildLookup_EURegion(t *testing.T) {
	h := &ClusterHandler{}

	lookup, err := h.BuildLookup("eu-central-1", nil)
	if err != nil {
		t.Fatalf("BuildLookup: %v", err)
	}
	if lookup.Attributes["usagetype"] != "EUC1-AmazonEKS-Hours:perCluster" {
		t.Errorf("usagetype = %q, want EUC1-AmazonEKS-Hours:perCluster", lookup.Attributes["usagetype"])
	}
}

func TestClusterHandler_BuildLookup_UnknownRegion(t *testing.T) {
	h := &ClusterHandler{}

	lookup, err := h.BuildLookup("xx-unknown-1", nil)
	if err != nil {
		t.Fatalf("BuildLookup: %v", err)
	}
	// Falls back to USE1 prefix
	if lookup.Attributes["usagetype"] != "USE1-AmazonEKS-Hours:perCluster" {
		t.Errorf("usagetype = %q, want USE1 fallback", lookup.Attributes["usagetype"])
	}
}

func TestClusterHandler_CalculateCost_FromAPI(t *testing.T) {
	h := &ClusterHandler{}

	price := &pricing.Price{OnDemandUSD: 0.10}
	hourly, monthly := h.CalculateCost(price, nil, "", nil)
	if hourly != 0.10 {
		t.Errorf("hourly = %v, want 0.10", hourly)
	}
	if monthly != 0.10*aws.HoursPerMonth {
		t.Errorf("monthly = %v, want %v", monthly, 0.10*aws.HoursPerMonth)
	}
}

func TestClusterHandler_CalculateCost_Fallback(t *testing.T) {
	h := &ClusterHandler{}

	// nil price -> fallback to default
	hourly, monthly := h.CalculateCost(nil, nil, "", nil)
	if hourly != DefaultClusterHourlyCost {
		t.Errorf("hourly = %v, want %v", hourly, DefaultClusterHourlyCost)
	}
	if monthly != DefaultClusterHourlyCost*aws.HoursPerMonth {
		t.Errorf("monthly = %v, want %v", monthly, DefaultClusterHourlyCost*aws.HoursPerMonth)
	}

	// zero price -> fallback
	hourly2, _ := h.CalculateCost(&pricing.Price{OnDemandUSD: 0}, nil, "", nil)
	if hourly2 != DefaultClusterHourlyCost {
		t.Errorf("hourly with zero price = %v, want fallback %v", hourly2, DefaultClusterHourlyCost)
	}
}

func TestClusterHandler_Describe(t *testing.T) {
	h := &ClusterHandler{}

	tests := []struct {
		name       string
		attrs      map[string]any
		wantKeys   map[string]string
		wantAbsent []string
	}{
		{
			name:       "nil attrs",
			attrs:      nil,
			wantAbsent: []string{"version"},
		},
		{
			name:       "empty attrs",
			attrs:      map[string]any{},
			wantAbsent: []string{"version"},
		},
		{
			name: "with version",
			attrs: map[string]any{
				"version": "1.28",
			},
			wantKeys: map[string]string{"version": "1.28"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.Describe(nil, tt.attrs)

			for k, v := range tt.wantKeys {
				if result[k] != v {
					t.Errorf("Describe()[%q] = %q, want %q", k, result[k], v)
				}
			}
			for _, k := range tt.wantAbsent {
				if _, ok := result[k]; ok {
					t.Errorf("Describe() should not contain key %q", k)
				}
			}
		})
	}
}
