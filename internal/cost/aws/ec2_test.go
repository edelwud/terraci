package aws

import (
	"testing"

	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestEC2InstanceHandler_ServiceCode(t *testing.T) {
	h := &EC2InstanceHandler{}
	if h.ServiceCode() != pricing.ServiceEC2 {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceEC2)
	}
}

func TestEC2InstanceHandler_BuildLookup(t *testing.T) {
	h := &EC2InstanceHandler{}

	tests := []struct {
		name        string
		region      string
		attrs       map[string]interface{}
		wantErr     bool
		wantType    string
		wantTenancy string
	}{
		{
			name:   "basic instance",
			region: "us-east-1",
			attrs: map[string]interface{}{
				"instance_type": "t3.micro",
			},
			wantErr:     false,
			wantType:    "t3.micro",
			wantTenancy: "Shared",
		},
		{
			name:   "dedicated tenancy",
			region: "eu-central-1",
			attrs: map[string]interface{}{
				"instance_type": "m5.large",
				"tenancy":       "dedicated",
			},
			wantErr:     false,
			wantType:    "m5.large",
			wantTenancy: "Dedicated",
		},
		{
			name:    "missing instance_type",
			region:  "us-east-1",
			attrs:   map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookup, err := h.BuildLookup(tt.region, tt.attrs)

			if tt.wantErr {
				if err == nil {
					t.Error("BuildLookup should return error")
				}
				return
			}

			if err != nil {
				t.Fatalf("BuildLookup returned error: %v", err)
			}

			if lookup.Attributes["instanceType"] != tt.wantType {
				t.Errorf("instanceType = %q, want %q", lookup.Attributes["instanceType"], tt.wantType)
			}

			if lookup.Attributes["tenancy"] != tt.wantTenancy {
				t.Errorf("tenancy = %q, want %q", lookup.Attributes["tenancy"], tt.wantTenancy)
			}
		})
	}
}

func TestEC2InstanceHandler_CalculateCost(t *testing.T) {
	h := &EC2InstanceHandler{}

	price := &pricing.Price{
		OnDemandUSD: 0.10, // $0.10/hour
	}

	hourly, monthly := h.CalculateCost(price, nil)

	if hourly != 0.10 {
		t.Errorf("hourly = %v, want %v", hourly, 0.10)
	}

	expectedMonthly := 0.10 * 730
	if monthly != expectedMonthly {
		t.Errorf("monthly = %v, want %v", monthly, expectedMonthly)
	}
}

func TestEBSVolumeHandler_BuildLookup(t *testing.T) {
	h := &EBSVolumeHandler{}

	tests := []struct {
		name           string
		attrs          map[string]interface{}
		wantVolumeType string
	}{
		{
			name:           "default gp2",
			attrs:          map[string]interface{}{},
			wantVolumeType: "General Purpose",
		},
		{
			name: "explicit gp3",
			attrs: map[string]interface{}{
				"type": "gp3",
			},
			wantVolumeType: "General Purpose",
		},
		{
			name: "io1",
			attrs: map[string]interface{}{
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

func TestEBSVolumeHandler_CalculateCost(t *testing.T) {
	h := &EBSVolumeHandler{}

	price := &pricing.Price{
		OnDemandUSD: 0.10, // $0.10 per GB-month
	}

	tests := []struct {
		name            string
		attrs           map[string]interface{}
		expectedMonthly float64
	}{
		{
			name: "default 8GB",
			attrs: map[string]interface{}{
				"type": "gp2",
			},
			expectedMonthly: 0.10 * 8,
		},
		{
			name: "100GB volume",
			attrs: map[string]interface{}{
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

func TestNATGatewayHandler_CalculateCost(t *testing.T) {
	h := &NATGatewayHandler{}

	// With price from lookup
	price := &pricing.Price{
		OnDemandUSD: 0.045,
	}

	hourly, monthly := h.CalculateCost(price, nil)

	if hourly != 0.045 {
		t.Errorf("hourly = %v, want %v", hourly, 0.045)
	}

	expectedMonthly := 0.045 * 730
	if monthly != expectedMonthly {
		t.Errorf("monthly = %v, want %v", monthly, expectedMonthly)
	}

	// Without price (fallback)
	hourly, _ = h.CalculateCost(&pricing.Price{OnDemandUSD: 0}, nil)
	if hourly != 0.045 {
		t.Errorf("fallback hourly = %v, want %v", hourly, 0.045)
	}
}
