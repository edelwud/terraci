package awskit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/caarlos0/log"

	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

const (
	// AWSPricingBaseURL is the base URL for AWS Price List Bulk API.
	AWSPricingBaseURL = "https://pricing.us-east-1.amazonaws.com"
	// DefaultTimeout for HTTP requests.
	DefaultTimeout = 5 * time.Minute
)

// awsPriceListOffer represents the structure of AWS price list JSON.
type awsPriceListOffer struct {
	FormatVersion   string                `json:"formatVersion"`
	Disclaimer      string                `json:"disclaimer"`
	OfferCode       string                `json:"offerCode"`
	Version         string                `json:"version"`
	PublicationDate string                `json:"publicationDate"`
	Products        map[string]awsProduct `json:"products"`
	Terms           awsTerms              `json:"terms"`
}

type awsProduct struct {
	SKU           string            `json:"sku"`
	ProductFamily string            `json:"productFamily"`
	Attributes    map[string]string `json:"attributes"`
}

type awsTerms struct {
	OnDemand map[string]map[string]awsPricingTerm `json:"OnDemand"`
	Reserved map[string]map[string]awsPricingTerm `json:"Reserved,omitempty"`
}

type awsPricingTerm struct {
	OfferTermCode   string                       `json:"offerTermCode"`
	SKU             string                       `json:"sku"`
	EffectiveDate   string                       `json:"effectiveDate"`
	PriceDimensions map[string]awsPriceDimension `json:"priceDimensions"`
	TermAttributes  map[string]string            `json:"termAttributes,omitempty"`
}

type awsPriceDimension struct {
	RateCode     string            `json:"rateCode"`
	Description  string            `json:"description"`
	BeginRange   string            `json:"beginRange"`
	EndRange     string            `json:"endRange"`
	Unit         string            `json:"unit"`
	PricePerUnit map[string]string `json:"pricePerUnit"`
	AppliesTo    []string          `json:"appliesTo,omitempty"`
}

// Fetcher downloads and parses AWS pricing data.
// Implements pricing.PriceFetcher.
type Fetcher struct {
	Client  *http.Client
	BaseURL string
}

// NewFetcher creates a new AWS pricing fetcher.
func NewFetcher() *Fetcher {
	return &Fetcher{
		Client: &http.Client{
			Timeout: DefaultTimeout,
		},
		BaseURL: AWSPricingBaseURL,
	}
}

// FetchRegionIndex downloads pricing for a specific service and region.
// Returns a compact PriceIndex suitable for caching.
func (f *Fetcher) FetchRegionIndex(ctx context.Context, service pricing.ServiceCode, region string) (*pricing.PriceIndex, error) {
	url := f.buildRegionURL(service, region)
	log.WithField("service", string(service)).
		WithField("region", region).
		Debug("fetching pricing data")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := f.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch pricing: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pricing API returned status %d", resp.StatusCode)
	}

	return f.parseToIndex(resp.Body, service, region)
}

func (f *Fetcher) buildRegionURL(service pricing.ServiceCode, region string) string {
	return fmt.Sprintf("%s/offers/v1.0/aws/%s/current/%s/index.json",
		f.BaseURL, service, region)
}

func (f *Fetcher) parseToIndex(r io.Reader, service pricing.ServiceCode, region string) (*pricing.PriceIndex, error) {
	var offer awsPriceListOffer
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&offer); err != nil {
		return nil, fmt.Errorf("decode pricing JSON: %w", err)
	}

	index := &pricing.PriceIndex{
		ServiceCode: service,
		Region:      region,
		Version:     offer.Version,
		UpdatedAt:   time.Now().UTC(),
		Products:    make(map[string]pricing.Price),
	}

	for sku, product := range offer.Products {
		skuTerms, ok := offer.Terms.OnDemand[sku]
		if !ok {
			continue
		}

		var priceUSD float64
		var unit string
		for _, term := range skuTerms {
			for _, dim := range term.PriceDimensions {
				if usd, ok := dim.PricePerUnit["USD"]; ok {
					if parsed, parseErr := strconv.ParseFloat(usd, 64); parseErr == nil {
						priceUSD = parsed
					}
					unit = dim.Unit
					break
				}
			}
			break
		}

		if priceUSD == 0 {
			continue
		}

		index.Products[sku] = pricing.Price{
			SKU:           sku,
			ProductFamily: product.ProductFamily,
			Attributes:    product.Attributes,
			OnDemandUSD:   priceUSD,
			Unit:          unit,
		}
	}

	log.WithField("service", string(service)).
		WithField("region", region).
		WithField("products", len(index.Products)).
		Debug("parsed pricing index")

	return index, nil
}
