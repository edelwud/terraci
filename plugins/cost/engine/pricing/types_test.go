package pricing

import "testing"

func TestRegionMapping(t *testing.T) {
	tests := []struct {
		regionCode string
		regionName string
	}{
		{"us-east-1", "US East (N. Virginia)"},
		{"eu-central-1", "EU (Frankfurt)"},
		{"ap-northeast-1", "Asia Pacific (Tokyo)"},
	}

	for _, tt := range tests {
		t.Run(tt.regionCode, func(t *testing.T) {
			name := RegionMapping[tt.regionCode]
			if name != tt.regionName {
				t.Errorf("RegionMapping[%q] = %q, want %q", tt.regionCode, name, tt.regionName)
			}
		})
	}
}

func TestRegionCodeMapping(t *testing.T) {
	// Verify reverse mapping works
	if RegionCodeMapping["US East (N. Virginia)"] != "us-east-1" {
		t.Error("RegionCodeMapping should map 'US East (N. Virginia)' to 'us-east-1'")
	}
}

func TestServiceCodes(t *testing.T) {
	tests := []struct {
		service ServiceCode
		value   string
	}{
		{ServiceEC2, "AmazonEC2"},
		{ServiceRDS, "AmazonRDS"},
		{ServiceS3, "AmazonS3"},
		{ServiceElastiCache, "AmazonElastiCache"},
		{ServiceEKS, "AmazonEKS"},
		{ServiceLambda, "AWSLambda"},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if string(tt.service) != tt.value {
				t.Errorf("ServiceCode = %q, want %q", tt.service, tt.value)
			}
		})
	}
}

func TestPriceIndex_LookupPrice(t *testing.T) {
	idx := &PriceIndex{
		ServiceCode: ServiceEC2,
		Region:      "us-east-1",
		Products: map[string]Price{
			"SKU1": {
				SKU:           "SKU1",
				ProductFamily: "Compute Instance",
				Attributes: map[string]string{
					"instanceType": "t3.micro",
					"location":     "US East (N. Virginia)",
				},
				OnDemandUSD: 0.0104,
			},
			"SKU2": {
				SKU:           "SKU2",
				ProductFamily: "Compute Instance",
				Attributes: map[string]string{
					"instanceType": "m5.large",
					"location":     "US East (N. Virginia)",
				},
				OnDemandUSD: 0.096,
			},
		},
	}

	tests := []struct {
		name    string
		lookup  PriceLookup
		wantSKU string
		wantErr bool
	}{
		{
			name: "find t3.micro",
			lookup: PriceLookup{
				ProductFamily: "Compute Instance",
				Attributes: map[string]string{
					"instanceType": "t3.micro",
				},
			},
			wantSKU: "SKU1",
		},
		{
			name: "find m5.large",
			lookup: PriceLookup{
				ProductFamily: "Compute Instance",
				Attributes: map[string]string{
					"instanceType": "m5.large",
				},
			},
			wantSKU: "SKU2",
		},
		{
			name: "not found",
			lookup: PriceLookup{
				ProductFamily: "Compute Instance",
				Attributes: map[string]string{
					"instanceType": "nonexistent",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price, err := idx.LookupPrice(tt.lookup)

			if tt.wantErr {
				if err == nil {
					t.Error("LookupPrice should return error")
				}
				return
			}

			if err != nil {
				t.Fatalf("LookupPrice returned error: %v", err)
			}

			if price.SKU != tt.wantSKU {
				t.Errorf("SKU = %q, want %q", price.SKU, tt.wantSKU)
			}
		})
	}
}

func TestMatchesLookup(t *testing.T) {
	price := Price{
		ProductFamily: "Compute Instance",
		Attributes: map[string]string{
			"instanceType": "t3.micro",
			"location":     "US East (N. Virginia)",
			"tenancy":      "Shared",
		},
	}

	tests := []struct {
		name   string
		lookup PriceLookup
		want   bool
	}{
		{
			name: "match all",
			lookup: PriceLookup{
				ProductFamily: "Compute Instance",
				Attributes: map[string]string{
					"instanceType": "t3.micro",
					"location":     "US East (N. Virginia)",
				},
			},
			want: true,
		},
		{
			name: "match partial",
			lookup: PriceLookup{
				ProductFamily: "Compute Instance",
				Attributes: map[string]string{
					"instanceType": "t3.micro",
				},
			},
			want: true,
		},
		{
			name: "wrong product family",
			lookup: PriceLookup{
				ProductFamily: "Storage",
				Attributes: map[string]string{
					"instanceType": "t3.micro",
				},
			},
			want: false,
		},
		{
			name: "wrong attribute",
			lookup: PriceLookup{
				ProductFamily: "Compute Instance",
				Attributes: map[string]string{
					"instanceType": "m5.large",
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesLookup(price, tt.lookup)
			if result != tt.want {
				t.Errorf("matchesLookup() = %v, want %v", result, tt.want)
			}
		})
	}
}
