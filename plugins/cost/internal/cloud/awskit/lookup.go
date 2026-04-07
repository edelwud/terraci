package awskit

import (
	"maps"

	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// PriceLookupSpec constructs PriceLookup objects with common defaults.
// It is a concrete builder helper — distinct from the handler.LookupBuilder interface.
type PriceLookupSpec struct {
	Service       pricing.ServiceID
	ProductFamily string
}

// Lookup creates a PriceLookup, automatically adding the resolved region name as "location".
// The caller's attrs map is not modified.
func (b *PriceLookupSpec) Lookup(region string, attrs map[string]string) *pricing.PriceLookup {
	merged := make(map[string]string, len(attrs)+1)
	maps.Copy(merged, attrs)
	merged["location"] = DefaultRuntime.ResolveRegionName(region)

	return &pricing.PriceLookup{
		ServiceID:     b.Service,
		Region:        region,
		ProductFamily: b.ProductFamily,
		Attributes:    merged,
	}
}
