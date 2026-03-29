package pricing

import "fmt"

// LookupPrice finds a price matching the given criteria in the index.
func (idx *PriceIndex) LookupPrice(lookup PriceLookup) (*Price, error) {
	for _, price := range idx.Products {
		if !matchesLookup(price, lookup) {
			continue
		}
		return &price, nil
	}
	return nil, fmt.Errorf("no matching price found for %+v", lookup)
}

// matchesLookup checks if a price matches the lookup criteria.
func matchesLookup(price Price, lookup PriceLookup) bool {
	if lookup.ProductFamily != "" && price.ProductFamily != lookup.ProductFamily {
		return false
	}

	for key, val := range lookup.Attributes {
		if price.Attributes[key] != val {
			return false
		}
	}

	return true
}
