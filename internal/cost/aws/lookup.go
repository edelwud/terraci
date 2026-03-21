package aws

import "github.com/edelwud/terraci/internal/cost/pricing"

// LookupBuilder constructs PriceLookup objects with common defaults.
type LookupBuilder struct {
	Service       pricing.ServiceCode
	ProductFamily string
}

// Build creates a PriceLookup, automatically adding the resolved region name as "location".
func (b *LookupBuilder) Build(region string, attrs map[string]string) *pricing.PriceLookup {
	if attrs == nil {
		attrs = make(map[string]string)
	}
	attrs["location"] = ResolveRegionName(region)

	return &pricing.PriceLookup{
		ServiceCode:   b.Service,
		Region:        region,
		ProductFamily: b.ProductFamily,
		Attributes:    attrs,
	}
}
