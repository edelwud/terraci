package elasticache

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// Backup storage fallback (used when API lookup unavailable).
const FallbackBackupStorageCostPerGBMonth = 0.085

// Data tiering fallback for r6gd/r7gd nodes.
const FallbackDataTieringCostPerGBMonth = 0.0125

type nodeCapacity struct {
	MemoryGiB float64
	SSDGiB    float64
}

// parseGiB extracts the numeric GiB value from AWS pricing attribute strings
// like "13.07 GiB", "75 GiB NVMe SSD", etc. Returns 0 if parsing fails.
func parseGiB(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "none") || s == "0" {
		return 0
	}
	// Take the first token (the number)
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return 0
	}
	v, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0
	}
	return v
}

func nodeCapacityFromPrice(price *pricing.Price) nodeCapacity {
	return nodeCapacity{
		MemoryGiB: nodeMemoryFromPrice(price),
		SSDGiB:    nodeSSDFromPrice(price),
	}
}

// nodeMemoryFromPrice extracts memory in GiB from the primary price's attributes.
// AWS pricing products include "memory" attribute like "13.07 GiB".
func nodeMemoryFromPrice(price *pricing.Price) float64 {
	if price == nil {
		return 0
	}
	return parseGiB(price.Attributes["memory"])
}

// nodeSSDFromPrice extracts SSD capacity in GiB from the primary price's attributes.
// AWS pricing products include "storage" attribute like "75 GiB NVMe SSD" or "None".
func nodeSSDFromPrice(price *pricing.Price) float64 {
	if price == nil {
		return 0
	}
	return parseGiB(price.Attributes["storage"])
}

func appendNodeCapacityDescribeFields(builder *awskit.DescribeBuilder, price *pricing.Price) {
	if builder == nil || price == nil {
		return
	}

	capacity := nodeCapacityFromPrice(price)
	if capacity.MemoryGiB > 0 {
		builder.String("memory_gib", fmt.Sprintf("%.2f", capacity.MemoryGiB))
	}
	if capacity.SSDGiB > 0 {
		builder.String("ssd_gib", fmt.Sprintf("%.0f", capacity.SSDGiB))
	}
}

// dataTieringCost calculates the SSD data tiering cost for nodes with local SSD.
// SSD capacity is extracted from the price's "storage" attribute.
func dataTieringCost(runtime *awskit.Runtime, price *pricing.Price, index *pricing.PriceIndex, region string, nodeCount int) float64 {
	ssdGB := nodeCapacityFromPrice(price).SSDGiB
	if ssdGB == 0 {
		return 0
	}

	costPerGB := FallbackDataTieringCostPerGBMonth
	if index != nil && region != "" {
		if looked, found := lookupElastiCachePrice(runtime, index, region, "Cache Storage", runtime.ResolveUsagePrefix(region)+"-DataTiering:StorageUsage"); found {
			costPerGB = looked
		}
	}

	return ssdGB * float64(nodeCount) * costPerGB
}

// backupStorageCost calculates backup storage cost.
// Memory size is extracted from the price's "memory" attribute.
// AWS provides free backup storage equal to the total cache memory.
// Additional backup is charged per GB-month.
func backupStorageCost(runtime *awskit.Runtime, price *pricing.Price, index *pricing.PriceIndex, region string, nodeCount, snapshotRetention int) float64 {
	memGB := nodeCapacityFromPrice(price).MemoryGiB
	if memGB == 0 {
		return 0
	}

	totalCacheGB := memGB * float64(nodeCount)
	// Estimate total backup size: one snapshot per day × retention days.
	// Free tier = total cache size (one snapshot).
	totalBackupGB := totalCacheGB * float64(snapshotRetention)
	chargeableGB := totalBackupGB - totalCacheGB
	if chargeableGB <= 0 {
		return 0
	}

	costPerGB := FallbackBackupStorageCostPerGBMonth
	if index != nil && region != "" {
		if looked, found := lookupElastiCachePrice(runtime, index, region, "Storage Snapshot", runtime.ResolveUsagePrefix(region)+"-BackupUsage"); found {
			costPerGB = looked
		}
	}

	return chargeableGB * costPerGB
}

func nodeStorageAddOnMonthlyCost(runtime *awskit.Runtime, price *pricing.Price, index *pricing.PriceIndex, region string, nodeCount, snapshotRetention int) float64 {
	monthly := dataTieringCost(runtime, price, index, region, nodeCount)
	if snapshotRetention > 0 {
		monthly += backupStorageCost(runtime, price, index, region, nodeCount, snapshotRetention)
	}
	return monthly
}

// lookupElastiCachePrice finds a price in the index by product family and usagetype.
func lookupElastiCachePrice(runtime *awskit.Runtime, index *pricing.PriceIndex, region, productFamily, usagetype string) (float64, bool) {
	if index == nil {
		return 0, false
	}
	lookup := pricing.PriceLookup{
		ProductFamily: productFamily,
		Attributes: map[string]string{
			"location":  runtime.ResolveRegionName(region),
			"usagetype": usagetype,
		},
	}
	p, err := index.LookupPrice(lookup)
	if err != nil {
		return 0, false
	}
	return p.OnDemandUSD, true
}
