package pricing

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildRegionURL(t *testing.T) {
	f := NewFetcher()

	tests := []struct {
		service ServiceCode
		region  string
		want    string
	}{
		{ServiceEC2, "us-east-1", "https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/AmazonEC2/current/us-east-1/index.json"},
		{ServiceRDS, "eu-west-1", "https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/AmazonRDS/current/eu-west-1/index.json"},
		{ServiceS3, "ap-northeast-1", "https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/AmazonS3/current/ap-northeast-1/index.json"},
	}

	for _, tt := range tests {
		t.Run(string(tt.service)+"_"+tt.region, func(t *testing.T) {
			got := f.buildRegionURL(tt.service, tt.region)
			if got != tt.want {
				t.Errorf("buildRegionURL(%s, %s) = %q, want %q", tt.service, tt.region, got, tt.want)
			}
		})
	}
}

func TestParseToIndex(t *testing.T) {
	offerJSON := `{
		"formatVersion": "v1.0",
		"disclaimer": "test",
		"offerCode": "AmazonEC2",
		"version": "20240101",
		"publicationDate": "2024-01-01",
		"products": {
			"SKU1": {
				"sku": "SKU1",
				"productFamily": "Compute Instance",
				"attributes": {
					"instanceType": "t3.micro",
					"location": "US East (N. Virginia)",
					"operatingSystem": "Linux",
					"tenancy": "Shared"
				}
			},
			"SKU2": {
				"sku": "SKU2",
				"productFamily": "Compute Instance",
				"attributes": {
					"instanceType": "m5.large",
					"location": "US East (N. Virginia)"
				}
			},
			"SKU_FREE": {
				"sku": "SKU_FREE",
				"productFamily": "Compute Instance",
				"attributes": {
					"instanceType": "free-tier"
				}
			}
		},
		"terms": {
			"OnDemand": {
				"SKU1": {
					"SKU1.TERM1": {
						"offerTermCode": "JRTCKXETXF",
						"sku": "SKU1",
						"effectiveDate": "2024-01-01",
						"priceDimensions": {
							"SKU1.TERM1.DIM1": {
								"rateCode": "SKU1.TERM1.DIM1",
								"description": "per hour",
								"unit": "Hrs",
								"pricePerUnit": {"USD": "0.0104"}
							}
						}
					}
				},
				"SKU2": {
					"SKU2.TERM1": {
						"offerTermCode": "JRTCKXETXF",
						"sku": "SKU2",
						"effectiveDate": "2024-01-01",
						"priceDimensions": {
							"SKU2.TERM1.DIM1": {
								"rateCode": "SKU2.TERM1.DIM1",
								"description": "per hour",
								"unit": "Hrs",
								"pricePerUnit": {"USD": "0.096"}
							}
						}
					}
				},
				"SKU_FREE": {
					"SKU_FREE.TERM1": {
						"offerTermCode": "JRTCKXETXF",
						"sku": "SKU_FREE",
						"effectiveDate": "2024-01-01",
						"priceDimensions": {
							"SKU_FREE.TERM1.DIM1": {
								"rateCode": "SKU_FREE.TERM1.DIM1",
								"description": "free",
								"unit": "Hrs",
								"pricePerUnit": {"USD": "0.0000000000"}
							}
						}
					}
				}
			}
		}
	}`

	f := NewFetcher()
	idx, err := f.parseToIndex(strings.NewReader(offerJSON), ServiceEC2, "us-east-1")
	if err != nil {
		t.Fatalf("parseToIndex() error: %v", err)
	}

	if idx.ServiceCode != ServiceEC2 {
		t.Errorf("ServiceCode = %s, want %s", idx.ServiceCode, ServiceEC2)
	}
	if idx.Region != "us-east-1" {
		t.Errorf("Region = %s, want us-east-1", idx.Region)
	}
	if idx.Version != "20240101" {
		t.Errorf("Version = %s, want 20240101", idx.Version)
	}

	// Should have 2 products (SKU_FREE has price 0 and is skipped)
	if len(idx.Products) != 2 {
		t.Errorf("expected 2 products, got %d", len(idx.Products))
	}

	sku1, ok := idx.Products["SKU1"]
	if !ok {
		t.Fatal("expected SKU1 in products")
	}
	if sku1.OnDemandUSD != 0.0104 {
		t.Errorf("SKU1 OnDemandUSD = %v, want 0.0104", sku1.OnDemandUSD)
	}
	if sku1.Unit != "Hrs" {
		t.Errorf("SKU1 Unit = %q, want %q", sku1.Unit, "Hrs")
	}
	if sku1.ProductFamily != "Compute Instance" {
		t.Errorf("SKU1 ProductFamily = %q, want %q", sku1.ProductFamily, "Compute Instance")
	}
	if sku1.Attributes["instanceType"] != "t3.micro" {
		t.Errorf("SKU1 instanceType = %q, want %q", sku1.Attributes["instanceType"], "t3.micro")
	}

	sku2, ok := idx.Products["SKU2"]
	if !ok {
		t.Fatal("expected SKU2 in products")
	}
	if sku2.OnDemandUSD != 0.096 {
		t.Errorf("SKU2 OnDemandUSD = %v, want 0.096", sku2.OnDemandUSD)
	}
}

func TestParseToIndex_InvalidJSON(t *testing.T) {
	f := NewFetcher()
	_, err := f.parseToIndex(strings.NewReader("not json"), ServiceEC2, "us-east-1")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseToIndex_EmptyProducts(t *testing.T) {
	offerJSON := `{
		"formatVersion": "v1.0",
		"offerCode": "AmazonEC2",
		"version": "v1",
		"products": {},
		"terms": {"OnDemand": {}}
	}`

	f := NewFetcher()
	idx, err := f.parseToIndex(strings.NewReader(offerJSON), ServiceEC2, "us-east-1")
	if err != nil {
		t.Fatalf("parseToIndex() error: %v", err)
	}
	if len(idx.Products) != 0 {
		t.Errorf("expected 0 products, got %d", len(idx.Products))
	}
}

func TestNewFetcher(t *testing.T) {
	f := NewFetcher()
	if f == nil {
		t.Fatal("NewFetcher returned nil")
	}
	if f.client == nil {
		t.Error("client is nil")
	}
	if f.baseURL != AWSPricingBaseURL {
		t.Errorf("baseURL = %q, want %q", f.baseURL, AWSPricingBaseURL)
	}
}

func TestMatchesLookup_EmptyProductFamily(t *testing.T) {
	price := Price{
		ProductFamily: "Compute Instance",
		Attributes: map[string]string{
			"instanceType": "t3.micro",
		},
	}

	// Empty product family should match any
	lookup := PriceLookup{
		ProductFamily: "",
		Attributes: map[string]string{
			"instanceType": "t3.micro",
		},
	}

	if !matchesLookup(price, lookup) {
		t.Error("matchesLookup should match when ProductFamily is empty")
	}
}

func TestMatchesLookup_EmptyAttributes(t *testing.T) {
	price := Price{
		ProductFamily: "Compute Instance",
		Attributes: map[string]string{
			"instanceType": "t3.micro",
		},
	}

	// Empty attributes should match any product of the same family
	lookup := PriceLookup{
		ProductFamily: "Compute Instance",
		Attributes:    map[string]string{},
	}

	if !matchesLookup(price, lookup) {
		t.Error("matchesLookup should match when Attributes is empty")
	}
}

func TestMatchesLookup_MissingAttribute(t *testing.T) {
	price := Price{
		ProductFamily: "Compute Instance",
		Attributes: map[string]string{
			"instanceType": "t3.micro",
		},
	}

	// Lookup requires attribute not present in price
	lookup := PriceLookup{
		ProductFamily: "Compute Instance",
		Attributes: map[string]string{
			"nonExistentKey": "value",
		},
	}

	if matchesLookup(price, lookup) {
		t.Error("matchesLookup should not match when required attribute is missing")
	}
}

func TestFetchRegionIndex(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{
			"formatVersion": "v1.0",
			"offerCode": "AmazonEC2",
			"version": "test-v1",
			"products": {
				"SKU1": {
					"sku": "SKU1",
					"productFamily": "Compute Instance",
					"attributes": {"instanceType": "t3.micro"}
				}
			},
			"terms": {
				"OnDemand": {
					"SKU1": {
						"SKU1.T1": {
							"offerTermCode": "JRTCKXETXF",
							"sku": "SKU1",
							"priceDimensions": {
								"SKU1.T1.D1": {
									"unit": "Hrs",
									"pricePerUnit": {"USD": "0.0104"}
								}
							}
						}
					}
				}
			}
		}`)
	}))
	defer ts.Close()

	f := &Fetcher{client: ts.Client(), baseURL: ts.URL}
	idx, err := f.FetchRegionIndex(context.Background(), ServiceEC2, "us-east-1")
	if err != nil {
		t.Fatalf("FetchRegionIndex: %v", err)
	}
	if idx.Version != "test-v1" {
		t.Errorf("Version = %q, want 'test-v1'", idx.Version)
	}
	if len(idx.Products) != 1 {
		t.Errorf("expected 1 product, got %d", len(idx.Products))
	}
}

func TestFetchRegionIndex_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	f := &Fetcher{client: ts.Client(), baseURL: ts.URL}
	_, err := f.FetchRegionIndex(context.Background(), ServiceEC2, "us-east-1")
	if err == nil {
		t.Error("expected error for non-200 status")
	}
}

func TestFetchRegionIndex_InvalidBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "not json")
	}))
	defer ts.Close()

	f := &Fetcher{client: ts.Client(), baseURL: ts.URL}
	_, err := f.FetchRegionIndex(context.Background(), ServiceEC2, "us-east-1")
	if err == nil {
		t.Error("expected error for invalid JSON body")
	}
}
