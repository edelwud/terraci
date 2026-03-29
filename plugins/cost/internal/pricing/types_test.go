package pricing

import "testing"

var (
	typesAWSProviderID         = "aws"
	typesAWSServiceEC2         = ServiceID{Provider: typesAWSProviderID, Name: "AmazonEC2"}
	typesAWSServiceRDS         = ServiceID{Provider: typesAWSProviderID, Name: "AmazonRDS"}
	typesAWSServiceS3          = ServiceID{Provider: typesAWSProviderID, Name: "AmazonS3"}
	typesAWSServiceElastiCache = ServiceID{Provider: typesAWSProviderID, Name: "AmazonElastiCache"}
	typesAWSServiceEKS         = ServiceID{Provider: typesAWSProviderID, Name: "AmazonEKS"}
	typesAWSServiceLambda      = ServiceID{Provider: typesAWSProviderID, Name: "AWSLambda"}
)

func TestServiceIDs(t *testing.T) {
	tests := []struct {
		service  ServiceID
		provider string
		name     string
	}{
		{typesAWSServiceEC2, typesAWSProviderID, "AmazonEC2"},
		{typesAWSServiceRDS, typesAWSProviderID, "AmazonRDS"},
		{typesAWSServiceS3, typesAWSProviderID, "AmazonS3"},
		{typesAWSServiceElastiCache, typesAWSProviderID, "AmazonElastiCache"},
		{typesAWSServiceEKS, typesAWSProviderID, "AmazonEKS"},
		{typesAWSServiceLambda, typesAWSProviderID, "AWSLambda"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.service.Provider != tt.provider {
				t.Errorf("Provider = %q, want %q", tt.service.Provider, tt.provider)
			}
			if tt.service.Name != tt.name {
				t.Errorf("Name = %q, want %q", tt.service.Name, tt.name)
			}
		})
	}
}

func TestPriceIndex_LookupPrice(t *testing.T) {
	idx := &PriceIndex{
		ServiceID: typesAWSServiceEC2,
		Region:    "us-east-1",
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
