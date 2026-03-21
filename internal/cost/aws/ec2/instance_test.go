package ec2

import (
	"testing"

	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestInstanceHandler_ServiceCode(t *testing.T) {
	h := &InstanceHandler{}
	if h.ServiceCode() != pricing.ServiceEC2 {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceEC2)
	}
}

func TestInstanceHandler_BuildLookup(t *testing.T) {
	h := &InstanceHandler{}

	tests := []struct {
		name        string
		region      string
		attrs       map[string]any
		wantErr     bool
		wantType    string
		wantTenancy string
	}{
		{
			name:   "basic instance",
			region: "us-east-1",
			attrs: map[string]any{
				"instance_type": "t3.micro",
			},
			wantErr:     false,
			wantType:    "t3.micro",
			wantTenancy: "Shared",
		},
		{
			name:   "dedicated tenancy",
			region: "eu-central-1",
			attrs: map[string]any{
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
			attrs:   map[string]any{},
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

func TestInstanceHandler_CalculateCost(t *testing.T) {
	h := &InstanceHandler{}

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
