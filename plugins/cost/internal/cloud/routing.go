package cloud

import (
	"context"
	"fmt"

	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// RoutingFetcher delegates pricing fetches to provider-specific fetchers.
// Used when multiple cloud providers are registered simultaneously.
type RoutingFetcher struct {
	Fetchers map[string]pricing.PriceFetcher // provider name → fetcher
}

// FetchRegionIndex tries each registered fetcher in order until one succeeds.
// This works because service codes are provider-specific (e.g., "AmazonEC2" is only
// meaningful to the AWS fetcher, GCP fetcher would return an error for it).
func (f *RoutingFetcher) FetchRegionIndex(ctx context.Context, service pricing.ServiceCode, region string) (*pricing.PriceIndex, error) {
	var lastErr error
	for name, fetcher := range f.Fetchers {
		idx, err := fetcher.FetchRegionIndex(ctx, service, region)
		if err == nil {
			return idx, nil
		}
		lastErr = fmt.Errorf("%s: %w", name, err)
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("no fetcher available for service %s in region %s", service, region)
}
