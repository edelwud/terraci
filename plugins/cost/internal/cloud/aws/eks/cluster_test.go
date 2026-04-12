package eks

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/definitiontest"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

func TestClusterHandler_Category(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompileTyped(ClusterSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))
	definitiontest.AssertCategory(t, def, resourcedef.CostCategoryStandard)
}

func TestClusterHandler_BuildLookup(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompileTyped(ClusterSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))

	definitiontest.RunLookupCases(t, def, []definitiontest.LookupCase{
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

	def := resourcespec.MustCompileTyped(ClusterSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))

	price := &pricing.Price{OnDemandUSD: 0.10}
	hourly, monthly, ok := def.CalculateStandardCost(price, nil, "", nil)
	if !ok {
		t.Fatal("CalculateStandardCost() ok = false, want true")
	}
	if hourly != 0.10 {
		t.Errorf("hourly = %v, want 0.10", hourly)
	}
	if monthly != 0.10*costutil.HoursPerMonth {
		t.Errorf("monthly = %v, want %v", monthly, 0.10*costutil.HoursPerMonth)
	}
}

func TestClusterHandler_CalculateCost_Fallback(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompileTyped(ClusterSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))

	// nil price -> fallback to default
	hourly, monthly, ok := def.CalculateStandardCost(nil, nil, "", nil)
	if !ok {
		t.Fatal("CalculateStandardCost() ok = false, want true")
	}
	if hourly != DefaultClusterHourlyCost {
		t.Errorf("hourly = %v, want %v", hourly, DefaultClusterHourlyCost)
	}
	if monthly != DefaultClusterHourlyCost*costutil.HoursPerMonth {
		t.Errorf("monthly = %v, want %v", monthly, DefaultClusterHourlyCost*costutil.HoursPerMonth)
	}

	// zero price -> fallback
	hourly2, _, ok2 := def.CalculateStandardCost(&pricing.Price{OnDemandUSD: 0}, nil, "", nil)
	if !ok2 {
		t.Fatal("CalculateStandardCost() ok = false, want true")
	}
	if hourly2 != DefaultClusterHourlyCost {
		t.Errorf("hourly with zero price = %v, want fallback %v", hourly2, DefaultClusterHourlyCost)
	}
}

func TestClusterHandler_Describe(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompileTyped(ClusterSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))

	definitiontest.RunDescribeCases(t, def, []definitiontest.DescribeCase{
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
