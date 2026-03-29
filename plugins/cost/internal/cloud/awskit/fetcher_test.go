package awskit

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

func TestBuildRegionURL(t *testing.T) {
	f := NewFetcher()

	tests := []struct {
		service pricing.ServiceID
		region  string
		want    string
	}{
		{MustService(ServiceKeyEC2), "us-east-1", "https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/AmazonEC2/current/us-east-1/index.json"},
		{MustService(ServiceKeyRDS), "eu-west-1", "https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/AmazonRDS/current/eu-west-1/index.json"},
		{MustService(ServiceKeyS3), "ap-northeast-1", "https://pricing.us-east-1.amazonaws.com/offers/v1.0/aws/AmazonS3/current/ap-northeast-1/index.json"},
	}

	for _, tt := range tests {
		t.Run(tt.service.String()+"_"+tt.region, func(t *testing.T) {
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
	idx, err := f.parseToIndex(strings.NewReader(offerJSON), MustService(ServiceKeyEC2), "us-east-1")
	if err != nil {
		t.Fatalf("parseToIndex() error: %v", err)
	}

	if idx.ServiceID != MustService(ServiceKeyEC2) {
		t.Errorf("ServiceID = %s, want %s", idx.ServiceID, MustService(ServiceKeyEC2))
	}
	if idx.Region != "us-east-1" {
		t.Errorf("Region = %s, want us-east-1", idx.Region)
	}
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

	f := &Fetcher{Client: ts.Client(), BaseURL: ts.URL}
	idx, err := f.FetchRegionIndex(context.Background(), MustService(ServiceKeyEC2), "us-east-1")
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

	f := &Fetcher{Client: ts.Client(), BaseURL: ts.URL}
	_, err := f.FetchRegionIndex(context.Background(), MustService(ServiceKeyEC2), "us-east-1")
	if err == nil {
		t.Error("expected error for non-200 status")
	}
}

func TestNewFetcher(t *testing.T) {
	f := NewFetcher()
	if f == nil {
		t.Fatal("NewFetcher returned nil")
	}
	if f.Client == nil {
		t.Error("client is nil")
	}
	if f.BaseURL != AWSPricingBaseURL {
		t.Errorf("baseURL = %q, want %q", f.BaseURL, AWSPricingBaseURL)
	}
}
