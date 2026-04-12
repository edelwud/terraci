package eks

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

func TestClusterHandler_Category(t *testing.T) {
	t.Parallel()

	h := resourcespec.MustHandler(ClusterSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))
	handlertest.AssertCategory(t, h, handler.CostCategoryStandard)
}

func TestClusterHandler_BuildLookup(t *testing.T) {
	t.Parallel()

	h, ok := resourcespec.MustHandler(ClusterSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest)))).(handler.LookupBuilder)
	if !ok {
		t.Fatal("handler should implement LookupBuilder")
	}

	handlertest.RunLookupCases(t, h, []handlertest.LookupCase{
		{
			Name:   "us east region",
			Region: "us-east-1",
			Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
				tb.Helper()
				if lookup.ServiceID != awskit.MustService(awskit.ServiceKeyEKS) {
					tb.Errorf("ServiceID = %q, want %q", lookup.ServiceID, awskit.MustService(awskit.ServiceKeyEKS))
				}
				if lookup.Attributes["usagetype"] != "USE1-AmazonEKS-Hours:perCluster" {
					tb.Errorf("usagetype = %q, want USE1-AmazonEKS-Hours:perCluster", lookup.Attributes["usagetype"])
				}
			},
		},
		{
			Name:   "eu region",
			Region: "eu-central-1",
			Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
				tb.Helper()
				if lookup.Attributes["usagetype"] != "EUC1-AmazonEKS-Hours:perCluster" {
					tb.Errorf("usagetype = %q, want EUC1-AmazonEKS-Hours:perCluster", lookup.Attributes["usagetype"])
				}
			},
		},
		{
			Name:   "unknown region falls back",
			Region: "xx-unknown-1",
			Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
				tb.Helper()
				if lookup.Attributes["usagetype"] != "USE1-AmazonEKS-Hours:perCluster" {
					tb.Errorf("usagetype = %q, want USE1 fallback", lookup.Attributes["usagetype"])
				}
			},
		},
	})
}

func TestClusterHandler_CalculateCost_FromAPI(t *testing.T) {
	t.Parallel()

	h, ok := resourcespec.MustHandler(ClusterSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest)))).(handler.StandardCostHandler)
	if !ok {
		t.Fatal("handler should implement StandardCostHandler")
	}

	price := &pricing.Price{OnDemandUSD: 0.10}
	hourly, monthly := h.CalculateCost(price, nil, "", nil)
	if hourly != 0.10 {
		t.Errorf("hourly = %v, want 0.10", hourly)
	}
	if monthly != 0.10*handler.HoursPerMonth {
		t.Errorf("monthly = %v, want %v", monthly, 0.10*handler.HoursPerMonth)
	}
}

func TestClusterHandler_CalculateCost_Fallback(t *testing.T) {
	t.Parallel()

	h, ok := resourcespec.MustHandler(ClusterSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest)))).(handler.StandardCostHandler)
	if !ok {
		t.Fatal("handler should implement StandardCostHandler")
	}

	// nil price -> fallback to default
	hourly, monthly := h.CalculateCost(nil, nil, "", nil)
	if hourly != DefaultClusterHourlyCost {
		t.Errorf("hourly = %v, want %v", hourly, DefaultClusterHourlyCost)
	}
	if monthly != DefaultClusterHourlyCost*handler.HoursPerMonth {
		t.Errorf("monthly = %v, want %v", monthly, DefaultClusterHourlyCost*handler.HoursPerMonth)
	}

	// zero price -> fallback
	hourly2, _ := h.CalculateCost(&pricing.Price{OnDemandUSD: 0}, nil, "", nil)
	if hourly2 != DefaultClusterHourlyCost {
		t.Errorf("hourly with zero price = %v, want fallback %v", hourly2, DefaultClusterHourlyCost)
	}
}

func TestClusterHandler_Describe(t *testing.T) {
	t.Parallel()

	h, ok := resourcespec.MustHandler(ClusterSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest)))).(handler.Describer)
	if !ok {
		t.Fatal("handler should implement Describer")
	}

	handlertest.RunDescribeCases(t, h, []handlertest.DescribeCase{
		{
			Name:       "nil attrs",
			Attrs:      nil,
			WantAbsent: []string{"version"},
		},
		{
			Name:       "empty attrs",
			Attrs:      map[string]any{},
			WantAbsent: []string{"version"},
		},
		{
			Name: "with version",
			Attrs: map[string]any{
				"version": "1.28",
			},
			WantKeys: map[string]string{"version": "1.28"},
		},
	})
}

func TestParseClusterAttrs(t *testing.T) {
	t.Parallel()

	got := parseClusterAttrs(map[string]any{"version": "1.29"})
	if got.Version != "1.29" {
		t.Fatalf("parseClusterAttrs().Version = %q, want 1.29", got.Version)
	}
}
