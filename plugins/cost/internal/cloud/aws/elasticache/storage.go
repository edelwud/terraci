package elasticache

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// Backup storage fallback (used when API lookup unavailable).
const FallbackBackupStorageCostPerGBMonth = 0.085

// Data tiering fallback for r6gd/r7gd nodes.
const FallbackDataTieringCostPerGBMonth = 0.0125

// priceGiB extracts a GiB value from the price attributes (e.g., "memory", "storage").
func priceGiB(price *pricing.Price, key string) float64 {
	if price == nil {
		return 0
	}
	return awskit.ParseGiB(price.Attributes[key])
}
