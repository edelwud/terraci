package ec2

import (
	"testing"

	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestEBSHandler_BuildLookup(t *testing.T) {
	h := &EBSHandler{}

	tests := []struct {
		name           string
		attrs          map[string]any
		wantVolumeType string
	}{
		{
			name:           "default gp2",
			attrs:          map[string]any{},
			wantVolumeType: "General Purpose",
		},
		{
			name: "explicit gp3",
			attrs: map[string]any{
				"type": "gp3",
			},
			wantVolumeType: "General Purpose",
		},
		{
			name: "io1",
			attrs: map[string]any{
				"type": "io1",
			},
			wantVolumeType: "Provisioned IOPS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			_, monthly := h.CalculateCost(price, tt.attrs)

			if monthly != tt.expectedMonthly {
				t.Errorf("monthly = %v, want %v", monthly, tt.expectedMonthly)
			}
		})
	}
}

func TestEBSHandler_CalculateCost_IO1(t *testing.T) {
	h := &EBSHandler{}

	price := &pricing.Price{
		OnDemandUSD: 0.125, // io1 per GB-month
	}

	attrs := map[string]any{
		"type": aws.VolumeTypeIO1,
		"size": float64(100),
		"iops": float64(3000),
	}

	_, monthly := h.CalculateCost(price, attrs)

	expectedMonthly := 0.125*100 + 3000*DefaultIO1IOPSCostPerMonth
	if monthly != expectedMonthly {
		t.Errorf("monthly = %v, want %v", monthly, expectedMonthly)
	}
}

func TestEBSHandler_CalculateCost_GP3Throughput(t *testing.T) {
	h := &EBSHandler{}

	price := &pricing.Price{
		OnDemandUSD: 0.08, // gp3 per GB-month
	}

	attrs := map[string]any{
		"type":       aws.VolumeTypeGP3,
		"size":       float64(100),
		"throughput": float64(250),
	}

	_, monthly := h.CalculateCost(price, attrs)

	expectedMonthly := 0.08*100 + (250-DefaultGP3FreeThroughputMBps)*DefaultGP3ThroughputCostPerMB
	if monthly != expectedMonthly {
		t.Errorf("monthly = %v, want %v", monthly, expectedMonthly)
	}
}
