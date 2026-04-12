package ec2

import (
	"math"
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

const floatTolerance = 1e-9

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < floatTolerance
}

func TestEBSHandler_Contract(t *testing.T) {
	t.Parallel()

	category := handler.CostCategoryStandard
	handlertest.RunContractSuite(t, resourcespec.MustCompile(EBSSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest)))), handlertest.ContractSuite{
		Category: &category,
		LookupCases: []handlertest.LookupCase{
			{
				Name:   "default gp2",
				Region: "us-east-1",
				Attrs:  map[string]any{},
				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
					tb.Helper()
					if lookup.Attributes["volumeApiName"] != "gp2" {
						tb.Errorf("volumeApiName = %q, want %q", lookup.Attributes["volumeApiName"], "gp2")
					}
				},
			},
			{
				Name:   "explicit gp3",
				Region: "us-east-1",
				Attrs: map[string]any{
					"type": "gp3",
				},
				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
					tb.Helper()
					if lookup.Attributes["volumeApiName"] != "gp3" {
						tb.Errorf("volumeApiName = %q, want %q", lookup.Attributes["volumeApiName"], "gp3")
					}
				},
			},
			{
				Name:   "io1",
				Region: "us-east-1",
				Attrs: map[string]any{
					"type": "io1",
				},
				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
					tb.Helper()
					if lookup.Attributes["volumeApiName"] != "io1" {
						tb.Errorf("volumeApiName = %q, want %q", lookup.Attributes["volumeApiName"], "io1")
					}
				},
			},
		},
		DescribeCases: []handlertest.DescribeCase{
			{
				Name:       "nil attrs",
				Attrs:      nil,
				WantAbsent: []string{"volume_type", "size_gb", "iops", "throughput_mbps"},
			},
			{
				Name: "volume_type and size",
				Attrs: map[string]any{
					"type": "gp3",
					"size": float64(100),
				},
				WantKeys: map[string]string{
					"volume_type": "gp3",
					"size_gb":     "100",
				},
			},
			{
				Name: "all fields",
				Attrs: map[string]any{
					"type":       "gp3",
					"size":       float64(200),
					"iops":       float64(5000),
					"throughput": float64(300),
				},
				WantKeys: map[string]string{
					"volume_type":     "gp3",
					"size_gb":         "200",
					"iops":            "5000",
					"throughput_mbps": "300",
				},
			},
		},
	})
}

func TestParseEBSVolumeAttrs_ParsesStringNumbersAndDefaults(t *testing.T) {
	t.Parallel()

	got := parseEBSVolumeAttrs(map[string]any{
		"type":       "gp3",
		"size":       "100",
		"iops":       "4000",
		"throughput": "250",
	})
	if got.VolumeType != "gp3" || got.SizeGB != 100 || got.IOPS != 4000 || got.Throughput != 250 {
		t.Fatalf("parseEBSVolumeAttrs() = %+v, want parsed gp3/100/4000/250", got)
	}

	defaulted := parseEBSVolumeAttrs(map[string]any{})
	if defaulted.VolumeType != awskit.VolumeTypeGP2 || defaulted.SizeGB != defaultRootVolumeGB {
		t.Fatalf("parseEBSVolumeAttrs(default) = %+v, want gp2/%d", defaulted, defaultRootVolumeGB)
	}
	if defaulted.VolumeTypeSet || defaulted.SizeGBSet {
		t.Fatalf("parseEBSVolumeAttrs(default) = %+v, want defaults marked implicit", defaulted)
	}
}

func TestEBSHandler_CalculateCost(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(EBSSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))

	price := &pricing.Price{
		OnDemandUSD: 0.10, // $0.10 per GB-month
	}

	tests := []struct {
		name            string
		attrs           map[string]any
		expectedMonthly float64
	}{
		{
			name: "default 8GB",
			attrs: map[string]any{
				"type": "gp2",
			},
			expectedMonthly: 0.10 * 8,
		},
		{
			name: "100GB volume",
			attrs: map[string]any{
				"type": "gp2",
				"size": float64(100),
			},
			expectedMonthly: 0.10 * 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, monthly, ok := def.CalculateStandardCost(price, nil, "", tt.attrs)
			if !ok {
				t.Fatal("CalculateStandardCost should return ok=true")
			}

			if monthly != tt.expectedMonthly {
				t.Errorf("monthly = %v, want %v", monthly, tt.expectedMonthly)
			}
		})
	}
}

func TestEBSHandler_CalculateCost_IO1(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(EBSSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))

	price := &pricing.Price{
		OnDemandUSD: 0.125, // io1 per GB-month
	}

	attrs := map[string]any{
		"type": awskit.VolumeTypeIO1,
		"size": float64(100),
		"iops": float64(3000),
	}

	_, monthly, ok := def.CalculateStandardCost(price, nil, "", attrs)
	if !ok {
		t.Fatal("CalculateStandardCost should return ok=true")
	}

	expectedMonthly := 0.125*100 + 3000*FallbackIO1IOPSCostPerMonth
	if monthly != expectedMonthly {
		t.Errorf("monthly = %v, want %v", monthly, expectedMonthly)
	}
}

func TestEBSHandler_CalculateCost_GP3Throughput(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(EBSSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))

	price := &pricing.Price{
		OnDemandUSD: 0.08, // gp3 per GB-month
	}

	attrs := map[string]any{
		"type":       awskit.VolumeTypeGP3,
		"size":       float64(100),
		"throughput": float64(250),
	}

	_, monthly, ok := def.CalculateStandardCost(price, nil, "", attrs)
	if !ok {
		t.Fatal("CalculateStandardCost should return ok=true")
	}

	expectedMonthly := 0.08*100 + (250-DefaultGP3FreeThroughputMBps)*FallbackGP3ThroughputCostPerMB
	if monthly != expectedMonthly {
		t.Errorf("monthly = %v, want %v", monthly, expectedMonthly)
	}
}

func TestEBSHandler_CalculateCost_IO1_WithIndex(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(EBSSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))

	storagePrice := &pricing.Price{OnDemandUSD: 0.125}

	index := &pricing.PriceIndex{
		Products: map[string]pricing.Price{
			"iops-sku": {
				ProductFamily: "System Operation",
				Attributes: map[string]string{
					"usagetype": "USE1-EBS:VolumeP-IOPS.piops",
					"location":  "US East (N. Virginia)",
				},
				OnDemandUSD: 0.070, // different from fallback 0.065
			},
		},
	}

	attrs := map[string]any{
		"type": awskit.VolumeTypeIO1,
		"size": float64(100),
		"iops": float64(3000),
	}

	_, monthly, ok := def.CalculateStandardCost(storagePrice, index, "us-east-1", attrs)
	if !ok {
		t.Fatal("CalculateStandardCost should return ok=true")
	}

	expectedMonthly := 0.125*100 + 3000*0.070
	if !approxEqual(monthly, expectedMonthly) {
		t.Errorf("monthly = %v, want %v", monthly, expectedMonthly)
	}
}

func TestEBSHandler_CalculateCost_IO2(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(EBSSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))

	storagePrice := &pricing.Price{OnDemandUSD: 0.125}

	index := &pricing.PriceIndex{
		Products: map[string]pricing.Price{
			"io2-iops-sku": {
				ProductFamily: "System Operation",
				Attributes: map[string]string{
					"usagetype": "USE1-EBS:VolumeP-IOPS.io2",
					"location":  "US East (N. Virginia)",
				},
				OnDemandUSD: 0.072,
			},
		},
	}

	attrs := map[string]any{
		"type": awskit.VolumeTypeIO2,
		"size": float64(50),
		"iops": float64(5000),
	}

	_, monthly, ok := def.CalculateStandardCost(storagePrice, index, "us-east-1", attrs)
	if !ok {
		t.Fatal("CalculateStandardCost should return ok=true")
	}

	expectedMonthly := 0.125*50 + 5000*0.072
	if !approxEqual(monthly, expectedMonthly) {
		t.Errorf("monthly = %v, want %v", monthly, expectedMonthly)
	}
}

func TestEBSHandler_CalculateCost_GP3(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(EBSSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))

	storagePrice := &pricing.Price{OnDemandUSD: 0.08}

	index := &pricing.PriceIndex{
		Products: map[string]pricing.Price{
			"gp3-iops-sku": {
				ProductFamily: "System Operation",
				Attributes: map[string]string{
					"usagetype": "USE1-EBS:VolumeP-IOPS.gp3",
					"location":  "US East (N. Virginia)",
				},
				OnDemandUSD: 0.007, // different from fallback 0.006
			},
			"gp3-throughput-sku": {
				ProductFamily: "Provisioned Throughput",
				Attributes: map[string]string{
					"usagetype": "USE1-EBS:VolumeP-Throughput.gp3",
					"location":  "US East (N. Virginia)",
				},
				OnDemandUSD: 0.045, // different from fallback 0.040
			},
		},
	}

	attrs := map[string]any{
		"type":       awskit.VolumeTypeGP3,
		"size":       float64(100),
		"iops":       float64(4000),
		"throughput": float64(250),
	}

	_, monthly, ok := def.CalculateStandardCost(storagePrice, index, "us-east-1", attrs)
	if !ok {
		t.Fatal("CalculateStandardCost should return ok=true")
	}

	expectedMonthly := 0.08*100 +
		(4000-DefaultGP3FreeIOPS)*0.007 +
		(250-DefaultGP3FreeThroughputMBps)*0.045
	if monthly != expectedMonthly {
		t.Errorf("monthly = %v, want %v", monthly, expectedMonthly)
	}
}

func TestEBSHandler_CalculateCost_FallbackOnMissingProduct(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(EBSSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))

	storagePrice := &pricing.Price{OnDemandUSD: 0.125}

	// Empty index — no IOPS products
	index := &pricing.PriceIndex{
		Products: map[string]pricing.Price{},
	}

	attrs := map[string]any{
		"type": awskit.VolumeTypeIO1,
		"size": float64(100),
		"iops": float64(3000),
	}

	_, monthly, ok := def.CalculateStandardCost(storagePrice, index, "us-east-1", attrs)
	if !ok {
		t.Fatal("CalculateStandardCost should return ok=true")
	}

	// Should fall back to FallbackIO1IOPSCostPerMonth
	expectedMonthly := 0.125*100 + 3000*FallbackIO1IOPSCostPerMonth
	if monthly != expectedMonthly {
		t.Errorf("monthly = %v, want %v (expected fallback pricing)", monthly, expectedMonthly)
	}
}

func TestEBSHandler_CalculateCost_NilIndex(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(EBSSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))

	storagePrice := &pricing.Price{OnDemandUSD: 0.08}

	attrs := map[string]any{
		"type":       awskit.VolumeTypeGP3,
		"size":       float64(100),
		"iops":       float64(4000),
		"throughput": float64(250),
	}

	// nil index — same as CalculateStandardCost path
	_, monthly, ok := def.CalculateStandardCost(storagePrice, nil, "", attrs)
	if !ok {
		t.Fatal("CalculateStandardCost should return ok=true")
	}

	expectedMonthly := 0.08*100 +
		(4000-DefaultGP3FreeIOPS)*FallbackGP3IOPSCostPerMonth +
		(250-DefaultGP3FreeThroughputMBps)*FallbackGP3ThroughputCostPerMB
	if monthly != expectedMonthly {
		t.Errorf("monthly = %v, want %v", monthly, expectedMonthly)
	}
}
