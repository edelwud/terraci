package awskit

import (
	"maps"

	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// LookupBuilder assembles AWS pricing lookups without per-resource attribute-map boilerplate.
type LookupBuilder struct {
	runtime       *Runtime
	service       pricing.ServiceID
	productFamily string
	attrs         map[string]string
}

// NewLookupBuilder creates a lookup builder bound to this runtime's service catalog.
func (r *Runtime) NewLookupBuilder(serviceKey ServiceKey, productFamily string) LookupBuilder {
	return LookupBuilder{
		runtime:       r,
		service:       r.MustService(serviceKey),
		productFamily: productFamily,
		attrs:         make(map[string]string),
	}
}

// ProductFamily overrides the lookup product family.
func (b LookupBuilder) ProductFamily(value string) LookupBuilder {
	b.productFamily = value
	return b
}

// MatchString maps a string value through cases with a fallback.
func MatchString(value, fallback string, cases map[string]string) string {
	if mapped, ok := cases[value]; ok {
		return mapped
	}
	return fallback
}

// ProductFamilyMatch overrides the product family using a string match table.
func (b LookupBuilder) ProductFamilyMatch(value, fallback string, cases map[string]string) LookupBuilder {
	return b.ProductFamily(MatchString(value, fallback, cases))
}

// Attr adds one lookup attribute when the value is non-empty.
func (b LookupBuilder) Attr(key, value string) LookupBuilder {
	if value == "" {
		return b
	}
	if b.attrs == nil {
		b.attrs = make(map[string]string)
	}
	b.attrs[key] = value
	return b
}

// AttrIf adds one lookup attribute when condition is true and the value is non-empty.
func (b LookupBuilder) AttrIf(condition bool, key, value string) LookupBuilder {
	if !condition {
		return b
	}
	return b.Attr(key, value)
}

// AttrMatch adds one attribute using a string match table.
func (b LookupBuilder) AttrMatch(key, value, fallback string, cases map[string]string) LookupBuilder {
	return b.Attr(key, MatchString(value, fallback, cases))
}

// UsageType sets the region-derived AWS usage type.
func (b LookupBuilder) UsageType(region, suffix string) LookupBuilder {
	if suffix == "" {
		return b
	}
	return b.Attr("usagetype", b.runtime.ResolveUsagePrefix(region)+"-"+suffix)
}

// Build constructs the final pricing lookup.
func (b LookupBuilder) Build(region string) *pricing.PriceLookup {
	attrs := make(map[string]string, len(b.attrs))
	maps.Copy(attrs, b.attrs)
	return (&PriceLookupSpec{
		Service:       b.service,
		ProductFamily: b.productFamily,
	}).Lookup(region, attrs)
}
