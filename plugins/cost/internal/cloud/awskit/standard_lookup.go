package awskit

import (
	"errors"

	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// StandardLookupSpec describes a provider-scoped standard pricing lookup.
type StandardLookupSpec struct {
	Service       pricing.ServiceID
	ProductFamily string
	BuildAttrs    func(region string, attrs map[string]any) (map[string]string, error)
}

// Build constructs a standard PriceLookup using provider-owned service metadata.
func (s StandardLookupSpec) Build(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	if s.BuildAttrs == nil {
		return nil, errors.New("standard lookup attributes builder not configured")
	}

	lookupAttrs, err := s.BuildAttrs(region, attrs)
	if err != nil {
		return nil, err
	}

	lb := &PriceLookupSpec{
		Service:       s.Service,
		ProductFamily: s.ProductFamily,
	}
	return lb.Lookup(region, lookupAttrs), nil
}
