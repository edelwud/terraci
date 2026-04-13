package awskit

import (
	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// costMode determines how the base cost is calculated.
type costMode int

const (
	costModeHourly  costMode = iota // rate is per-hour
	costModePerUnit                 // rate is per-unit (monthly = rate × qty)
)

// RateResolver looks up a per-unit rate from the pricing index.
// Returns the rate and true if found, or zero and false to fall back.
type RateResolver func(index *pricing.PriceIndex, region string) (float64, bool)

// IndexRate returns a RateResolver that performs a secondary pricing lookup
// using the regional-prefix pattern (prefix-suffix, then unprefixed fallback).
func IndexRate(runtime *Runtime, productFamily, usageSuffix string) RateResolver {
	return func(index *pricing.PriceIndex, region string) (float64, bool) {
		if index == nil {
			return 0, false
		}

		location := runtime.ResolveRegionName(region)
		prefix := runtime.ResolveUsagePrefix(region)

		// Try with regional prefix: "USE1-EBS:VolumeP-IOPS.gp3"
		if p, err := index.LookupPrice(pricing.PriceLookup{
			ProductFamily: productFamily,
			Attributes: map[string]string{
				"location":  location,
				"usagetype": prefix + "-" + usageSuffix,
			},
		}); err == nil {
			return p.OnDemandUSD, true
		}

		// Fallback: unprefixed usagetype (us-east-1 quirk)
		if p, err := index.LookupPrice(pricing.PriceLookup{
			ProductFamily: productFamily,
			Attributes: map[string]string{
				"location":  location,
				"usagetype": usageSuffix,
			},
		}); err == nil {
			return p.OnDemandUSD, true
		}

		return 0, false
	}
}

// Charge describes a single cost component: (qty - freeTier) × rate.
// Create with NewCharge(qty), then chain FreeTier/Fixed/Rate/Fallback.
type Charge struct {
	qty      float64
	freeTier float64
	fixed    float64
	rate     RateResolver
	fallback float64
}

// NewCharge creates a charge for the given quantity of billable units.
func NewCharge(qty float64) Charge {
	return Charge{qty: qty}
}

// FreeTier subtracts a free allowance from the quantity.
func (c Charge) FreeTier(threshold float64) Charge {
	c.freeTier = threshold
	return c
}

// Fixed sets a constant per-unit rate.
func (c Charge) Fixed(rate float64) Charge {
	c.fixed = rate
	return c
}

// Rate sets a dynamic rate resolver (e.g. IndexRate).
func (c Charge) Rate(resolver RateResolver) Charge {
	c.rate = resolver
	return c
}

// Fallback sets the rate to use when the Rate resolver returns false.
func (c Charge) Fallback(rate float64) Charge {
	c.fallback = rate
	return c
}

// resolveRate determines the per-unit rate for a charge.
func (c Charge) resolveRate(index *pricing.PriceIndex, region string) float64 {
	if c.fixed != 0 {
		return c.fixed
	}
	if c.rate != nil {
		if r, ok := c.rate(index, region); ok {
			return r
		}
	}
	return c.fallback
}

// CostBuilder assembles a standard cost calculation using a fluent API.
// Create with NewCostBuilder(), configure base mode and charges,
// then call Calc() to compute hourly and monthly costs.
type CostBuilder struct {
	mode        costMode
	units       float64
	scale       float64
	fallback    float64
	hasFallback bool
	charges     []Charge
	matchKey    string
	matchCases  map[string][]Charge
	matchFb     []Charge
	hasMatch    bool
}

// NewCostBuilder creates a new cost builder.
func NewCostBuilder() *CostBuilder {
	return &CostBuilder{scale: 1}
}

// Hourly sets the base cost mode to per-hour pricing.
func (b *CostBuilder) Hourly() *CostBuilder {
	b.mode = costModeHourly
	return b
}

// PerUnit sets the base cost mode to per-unit pricing (monthly = rate × qty).
func (b *CostBuilder) PerUnit(qty float64) *CostBuilder {
	b.mode = costModePerUnit
	b.units = qty
	return b
}

// Scale multiplies the hourly rate by count (e.g. number of nodes).
func (b *CostBuilder) Scale(count float64) *CostBuilder {
	b.scale = count
	return b
}

// Fallback sets the hourly rate to use when the price lookup returns nil or zero.
func (b *CostBuilder) Fallback(rate float64) *CostBuilder {
	b.fallback = rate
	b.hasFallback = true
	return b
}

// Charge adds an unconditional charge evaluated via the Charge builder.
func (b *CostBuilder) Charge(c Charge) *CostBuilder {
	b.charges = append(b.charges, c)
	return b
}

// Match adds conditional charges selected by key.
// Looks up key in cases; if not found, applies fallback charges (may be nil).
func (b *CostBuilder) Match(key string, fallback []Charge, cases map[string][]Charge) *CostBuilder {
	b.matchKey = key
	b.matchCases = cases
	b.matchFb = fallback
	b.hasMatch = true
	return b
}

// Calc computes hourly and monthly costs from the configured builder state.
func (b *CostBuilder) Calc(price *pricing.Price, index *pricing.PriceIndex, region string) (hourly, monthly float64) {
	rate := b.resolveRate(price)
	if rate == 0 {
		return 0, 0
	}

	switch b.mode {
	case costModeHourly:
		hourly = rate * b.scale
		monthly = hourly * costutil.HoursPerMonth
	case costModePerUnit:
		monthly = rate * b.units
		hourly = monthly / costutil.HoursPerMonth
	}

	hasExtras := false
	for _, ch := range b.charges {
		qty := ch.qty - ch.freeTier
		if qty > 0 {
			monthly += qty * ch.resolveRate(index, region)
			hasExtras = true
		}
	}
	if b.hasMatch {
		monthly += calcCharges(b.matchKey, b.matchCases, b.matchFb, index, region)
		hasExtras = true
	}
	if hasExtras {
		hourly = monthly / costutil.HoursPerMonth
	}

	return hourly, monthly
}

// resolveRate extracts the base rate from the price or falls back.
func (b *CostBuilder) resolveRate(price *pricing.Price) float64 {
	if price != nil && price.OnDemandUSD > 0 {
		return price.OnDemandUSD
	}
	if b.hasFallback {
		return b.fallback
	}
	return 0
}

// calcCharges computes the total monthly cost from matched charges.
func calcCharges(key string, cases map[string][]Charge, fallback []Charge, index *pricing.PriceIndex, region string) float64 {
	charges, ok := cases[key]
	if !ok {
		charges = fallback
	}

	var total float64
	for _, ch := range charges {
		qty := ch.qty - ch.freeTier
		if qty <= 0 {
			continue
		}
		total += qty * ch.resolveRate(index, region)
	}
	return total
}
