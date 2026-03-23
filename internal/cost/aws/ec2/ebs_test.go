package ec2

import (
	"math"
	"testing"

	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestEBSHandler_Category(t *testing.T) {
	t.Parallel()

	h := &EBSHandler{}
	if h.Category() != aws.CostCategoryStandard {
		t.Errorf("Category() = %v, want CostCategoryStandard", h.Category())
	}
}

func TestEBSHandler_ServiceCode(t *testing.T) {
	t.Parallel()

	h := &EBSHandler{}
	if h.ServiceCode() != pricing.ServiceEC2 {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceEC2)
	}
}

func TestEBSHandler_Describe(t *testing.T) {
	t.Parallel()

	h := &EBSHandler{}

	tests := []struct {
		name       string
		attrs      map[string]any
		wantKeys   map[string]string
		wantAbsent []string
	}{
		{
			name:       "nil attrs",
			attrs:      nil,
			wantAbsent: []string{"volume_type", "size_gb", "iops", "throughput_mbps"},
		},
		{
			name: "volume_type and size",
			attrs: map[string]any{
				"type": "gp3",
				"size": float64(100),
			},
			wantKeys: map[string]string{
				"volume_type": "gp3",
				"size_gb":     "100",
			},
		},
		{
			name: "all fields",
			attrs: map[string]any{
				"type":       "gp3",
				"size":       float64(200),
				"iops":       float64(5000),
				"throughput": float64(300),
			},
			wantKeys: map[string]string{
				"volume_type":     "gp3",
				"size_gb":         "200",
				"iops":            "5000",
				"throughput_mbps": "300",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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

const floatTolerance = 1e-9

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < floatTolerance
}

func TestEBSHandler_BuildLookup(t *testing.T) {
	t.Parallel()

	h := &EBSHandler{}

	tests := []struct {
		name           string
		attrs          map[string]any
		wantVolumeType string
	}{
		{
			name:           "default gp2",
			attrs:          map[string]any{},
			wantVolumeType: "gp2",
		},
		{
			name: "explicit gp3",
			attrs: map[string]any{
				"type": "gp3",
			},
			wantVolumeType: "gp3",
		},
		{
			name: "io1",
			attrs: map[string]any{
				"type": "io1",
			},
			wantVolumeType: "io1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lookup, err := h.BuildLookup("us-east-1", tt.attrs)
			if err != nil {
				t.Fatalf("BuildLookup returned error: %v", err)
			}

			if lookup.Attributes["volumeApiName"] != tt.wantVolumeType {
				t.Errorf("volumeApiName = %q, want %q", lookup.Attributes["volumeApiName"], tt.wantVolumeType)
			}
		})
	}
}

func TestEBSHandler_CalculateCost(t *testing.T) {
	t.Parallel()

	h := &EBSHandler{}

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

			_, monthly := h.CalculateCost(price, nil, "", tt.attrs)

			if monthly != tt.expectedMonthly {
				t.Errorf("monthly = %v, want %v", monthly, tt.expectedMonthly)
			}
		})
	}
}

func TestEBSHandler_CalculateCost_IO1(t *testing.T) {
	t.Parallel()

	h := &EBSHandler{}

	price := &pricing.Price{
		OnDemandUSD: 0.125, // io1 per GB-month
	}

	attrs := map[string]any{
		"type": aws.VolumeTypeIO1,
		"size": float64(100),
		"iops": float64(3000),
	}

	_, monthly := h.CalculateCost(price, nil, "", attrs)

	expectedMonthly := 0.125*100 + 3000*FallbackIO1IOPSCostPerMonth
	if monthly != expectedMonthly {
		t.Errorf("monthly = %v, want %v", monthly, expectedMonthly)
	}
}

func TestEBSHandler_CalculateCost_GP3Throughput(t *testing.T) {
	t.Parallel()

	h := &EBSHandler{}

	price := &pricing.Price{
		OnDemandUSD: 0.08, // gp3 per GB-month
	}

	attrs := map[string]any{
		"type":       aws.VolumeTypeGP3,
		"size":       float64(100),
		"throughput": float64(250),
	}

	_, monthly := h.CalculateCost(price, nil, "", attrs)

	expectedMonthly := 0.08*100 + (250-DefaultGP3FreeThroughputMBps)*FallbackGP3ThroughputCostPerMB
	if monthly != expectedMonthly {
		t.Errorf("monthly = %v, want %v", monthly, expectedMonthly)
	}
}

func TestEBSHandler_CalculateCost_IO1_WithIndex(t *testing.T) {
	t.Parallel()

	h := &EBSHandler{}

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
		"type": aws.VolumeTypeIO1,
		"size": float64(100),
		"iops": float64(3000),
	}

	_, monthly := h.CalculateCost(storagePrice, index, "us-east-1", attrs)

	expectedMonthly := 0.125*100 + 3000*0.070
	if !approxEqual(monthly, expectedMonthly) {
		t.Errorf("monthly = %v, want %v", monthly, expectedMonthly)
	}
}

func TestEBSHandler_CalculateCost_IO2(t *testing.T) {
	t.Parallel()

	h := &EBSHandler{}

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
		"type": aws.VolumeTypeIO2,
		"size": float64(50),
		"iops": float64(5000),
	}

	_, monthly := h.CalculateCost(storagePrice, index, "us-east-1", attrs)

	expectedMonthly := 0.125*50 + 5000*0.072
	if !approxEqual(monthly, expectedMonthly) {
		t.Errorf("monthly = %v, want %v", monthly, expectedMonthly)
	}
}

func TestEBSHandler_CalculateCost_GP3(t *testing.T) {
	t.Parallel()

	h := &EBSHandler{}

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
		"type":       aws.VolumeTypeGP3,
		"size":       float64(100),
		"iops":       float64(4000),
		"throughput": float64(250),
	}

	_, monthly := h.CalculateCost(storagePrice, index, "us-east-1", attrs)

	expectedMonthly := 0.08*100 +
		(4000-DefaultGP3FreeIOPS)*0.007 +
		(250-DefaultGP3FreeThroughputMBps)*0.045
	if monthly != expectedMonthly {
		t.Errorf("monthly = %v, want %v", monthly, expectedMonthly)
	}
}

func TestEBSHandler_CalculateCost_FallbackOnMissingProduct(t *testing.T) {
	t.Parallel()

	h := &EBSHandler{}

	storagePrice := &pricing.Price{OnDemandUSD: 0.125}

	// Empty index — no IOPS products
	index := &pricing.PriceIndex{
		Products: map[string]pricing.Price{},
	}

	attrs := map[string]any{
		"type": aws.VolumeTypeIO1,
		"size": float64(100),
		"iops": float64(3000),
	}

	_, monthly := h.CalculateCost(storagePrice, index, "us-east-1", attrs)

	// Should fall back to FallbackIO1IOPSCostPerMonth
	expectedMonthly := 0.125*100 + 3000*FallbackIO1IOPSCostPerMonth
	if monthly != expectedMonthly {
		t.Errorf("monthly = %v, want %v (expected fallback pricing)", monthly, expectedMonthly)
	}
}

func TestEBSHandler_CalculateCost_NilIndex(t *testing.T) {
	t.Parallel()

	h := &EBSHandler{}

	storagePrice := &pricing.Price{OnDemandUSD: 0.08}

	attrs := map[string]any{
		"type":       aws.VolumeTypeGP3,
		"size":       float64(100),
		"iops":       float64(4000),
		"throughput": float64(250),
	}

	// nil index — same as CalculateCost path
	_, monthly := h.CalculateCost(storagePrice, nil, "", attrs)

	expectedMonthly := 0.08*100 +
		(4000-DefaultGP3FreeIOPS)*FallbackGP3IOPSCostPerMonth +
		(250-DefaultGP3FreeThroughputMBps)*FallbackGP3ThroughputCostPerMB
	if monthly != expectedMonthly {
		t.Errorf("monthly = %v, want %v", monthly, expectedMonthly)
	}
}
