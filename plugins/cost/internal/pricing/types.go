// Package pricing provides provider-neutral pricing types.
package pricing

import "time"

// ServiceID uniquely identifies a pricing service inside a specific cloud provider.
type ServiceID struct {
	Provider string `json:"provider"`
	Name     string `json:"name"`
}

// ServiceCatalog keeps provider-owned pricing services under stable keys.
type ServiceCatalog map[string]ServiceID

// ProviderManifest describes the provider runtime contract exposed to the cost engine.
type ProviderManifest struct {
	ID          string         `json:"id"`
	DisplayName string         `json:"display_name"`
	PriceSource string         `json:"price_source"`
	Services    ServiceCatalog `json:"services,omitempty"`
	Regions     RegionResolver `json:"-"`
}

// String returns a stable textual representation of the service identifier.
func (s ServiceID) String() string {
	if s.Provider == "" {
		return s.Name
	}
	return s.Provider + ":" + s.Name
}

// Service returns a registered service by its stable catalog key.
func (m ProviderManifest) Service(key string) (ServiceID, bool) {
	service, ok := m.Services[key]
	return service, ok
}

// MustService returns a registered service or panics if the catalog entry is missing.
func (m ProviderManifest) MustService(key string) ServiceID {
	service, ok := m.Service(key)
	if !ok {
		panic("pricing: provider service not registered: " + m.ID + "/" + key)
	}
	return service
}

// RegionResolver keeps provider-specific region naming and usage prefix rules.
type RegionResolver struct {
	LocationNames      map[string]string `json:"-"`
	UsagePrefixes      map[string]string `json:"-"`
	DefaultUsagePrefix string            `json:"-"`
}

// ResolveLocationName returns the pricing location name for a cloud region code.
// Falls back to the original region code when no mapping exists.
func (r RegionResolver) ResolveLocationName(region string) string {
	if name := r.LocationNames[region]; name != "" {
		return name
	}
	return region
}

// ResolveUsagePrefix returns the usage prefix for a cloud region code.
// Falls back to the configured default when no provider-specific mapping exists.
func (r RegionResolver) ResolveUsagePrefix(region string) string {
	if prefix := r.UsagePrefixes[region]; prefix != "" {
		return prefix
	}
	return r.DefaultUsagePrefix
}

// PriceIndex represents a compact pricing index for a service/region.
type PriceIndex struct {
	ServiceID  ServiceID         `json:"service_id"`
	Region     string            `json:"region"`
	Version    string            `json:"version"`
	UpdatedAt  time.Time         `json:"updated_at"`
	Products   map[string]Price  `json:"products"` // SKU -> Price
	Attributes map[string]string `json:"attributes,omitempty"`
}

// Price represents a single product price.
type Price struct {
	SKU           string            `json:"sku"`
	ProductFamily string            `json:"product_family"`
	Attributes    map[string]string `json:"attributes"`
	OnDemandUSD   float64           `json:"on_demand_usd"` // OnDemand hourly price in USD
	Unit          string            `json:"unit"`          // Hrs, GB-Mo, etc.
}

// PriceLookup represents criteria for finding a price.
type PriceLookup struct {
	ServiceID     ServiceID
	Region        string
	ProductFamily string
	Attributes    map[string]string
}

// isComplete checks that the index contains usable data (structural check, not TTL).
func (idx *PriceIndex) isComplete() bool {
	return idx != nil && idx.ServiceID.Name != "" && idx.Region != "" && len(idx.Products) > 0
}
