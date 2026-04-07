package ec2

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// EBS pricing fallbacks (used when API lookup unavailable).
const (
	FallbackIO1IOPSCostPerMonth    = 0.065
	FallbackIO2IOPSCostPerMonth    = 0.065
	FallbackGP3IOPSCostPerMonth    = 0.006
	FallbackGP3ThroughputCostPerMB = 0.040
)

// EBS product limits (not prices — these are AWS-defined free tiers).
const (
	DefaultGP3FreeIOPS           = 3000
	DefaultGP3FreeThroughputMBps = 125
)

// ebsVolumeAPIName maps Terraform volume types to AWS pricing volumeApiName.
// AWS API uses short names: "gp2", "gp3", "io1", "io2", "st1", "sc1", "standard".
var ebsVolumeAPIName = map[string]string{
	awskit.VolumeTypeGP2:      "gp2",
	awskit.VolumeTypeGP3:      "gp3",
	awskit.VolumeTypeIO1:      "io1",
	awskit.VolumeTypeIO2:      "io2",
	awskit.VolumeTypeST1:      "st1",
	awskit.VolumeTypeSC1:      "sc1",
	awskit.VolumeTypeStandard: "standard",
}

// EBSHandler handles aws_ebs_volume cost estimation.
type EBSHandler struct {
	awskit.RuntimeDeps
}

func (h *EBSHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *EBSHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	volumeType := handler.GetStringAttr(attrs, "type")
	if volumeType == "" {
		volumeType = awskit.VolumeTypeGP2
	}

	apiName := ebsVolumeAPIName[volumeType]
	if apiName == "" {
		apiName = "gp2"
	}

	runtime := h.RuntimeOrDefault()
	lb := &awskit.PriceLookupSpec{Service: runtime.MustService(awskit.ServiceKeyEC2), ProductFamily: "Storage"}
	return lb.Lookup(region, map[string]string{
		"volumeApiName": apiName,
	}), nil
}

func (h *EBSHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	return awskit.NewDescribeBuilder().
		String("volume_type", handler.GetStringAttr(attrs, "type")).
		Float("size_gb", handler.GetFloatAttr(attrs, "size"), "%.0f").
		Float("iops", handler.GetFloatAttr(attrs, "iops"), "%.0f").
		Float("throughput_mbps", handler.GetFloatAttr(attrs, "throughput"), "%.0f").
		Map()
}

// CalculateCost calculates EBS cost using the price index for
// secondary lookups (IOPS, throughput). Falls back to hardcoded values
// when the index is nil or the lookup fails.
func (h *EBSHandler) CalculateCost(price *pricing.Price, index *pricing.PriceIndex, region string, attrs map[string]any) (hourly, monthly float64) {
	size := handler.GetFloatAttr(attrs, "size")
	if size == 0 {
		size = 8
	}

	// Storage cost (per GB-month from primary API lookup)
	monthly = price.OnDemandUSD * size

	volumeType := handler.GetStringAttr(attrs, "type")

	// IOPS cost for io1/io2
	if volumeType == awskit.VolumeTypeIO1 || volumeType == awskit.VolumeTypeIO2 {
		iops := handler.GetFloatAttr(attrs, "iops")
		if iops > 0 {
			suffix := "piops"
			fallback := FallbackIO1IOPSCostPerMonth
			if volumeType == awskit.VolumeTypeIO2 {
				suffix = "io2"
				fallback = FallbackIO2IOPSCostPerMonth
			}
			iopsCost, ok := lookupEBSPrice(h.RuntimeOrDefault(), index, region, "System Operation", "EBS:VolumeP-IOPS."+suffix)
			if !ok {
				iopsCost = fallback
			}
			monthly += iops * iopsCost
		}
	}

	// gp3: IOPS cost above free tier (3000 free)
	if volumeType == awskit.VolumeTypeGP3 {
		iops := handler.GetFloatAttr(attrs, "iops")
		if iops > DefaultGP3FreeIOPS {
			iopsCost, ok := lookupEBSPrice(h.RuntimeOrDefault(), index, region, "System Operation", "EBS:VolumeP-IOPS.gp3")
			if !ok {
				iopsCost = FallbackGP3IOPSCostPerMonth
			}
			monthly += (iops - DefaultGP3FreeIOPS) * iopsCost
		}

		// gp3: throughput cost above free tier (125 MBps free)
		throughput := handler.GetFloatAttr(attrs, "throughput")
		if throughput > DefaultGP3FreeThroughputMBps {
			tpCost, ok := lookupEBSPrice(h.RuntimeOrDefault(), index, region, "Provisioned Throughput", "EBS:VolumeP-Throughput.gp3")
			if !ok {
				tpCost = FallbackGP3ThroughputCostPerMB
			}
			monthly += (throughput - DefaultGP3FreeThroughputMBps) * tpCost
		}
	}

	hourly = monthly / handler.HoursPerMonth
	return hourly, monthly
}

// lookupEBSPrice finds a price in the index by product family and usagetype.
// Tries with regional prefix first, then without (us-east-1 uses unprefixed usagetypes).
func lookupEBSPrice(runtime *awskit.Runtime, index *pricing.PriceIndex, region, productFamily, usageSuffix string) (float64, bool) {
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

	// Fallback: unprefixed usagetype (us-east-1 quirk): "EBS:VolumeP-IOPS.gp3"
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
