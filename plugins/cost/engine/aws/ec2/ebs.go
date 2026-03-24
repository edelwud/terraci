package ec2

import (
	"fmt"

	aws "github.com/edelwud/terraci/plugins/cost/engine/aws"
	"github.com/edelwud/terraci/plugins/cost/engine/pricing"
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
	aws.VolumeTypeGP2:      "gp2",
	aws.VolumeTypeGP3:      "gp3",
	aws.VolumeTypeIO1:      "io1",
	aws.VolumeTypeIO2:      "io2",
	aws.VolumeTypeST1:      "st1",
	aws.VolumeTypeSC1:      "sc1",
	aws.VolumeTypeStandard: "standard",
}

// EBSHandler handles aws_ebs_volume cost estimation.
type EBSHandler struct{}

func (h *EBSHandler) Category() aws.CostCategory { return aws.CostCategoryStandard }

func (h *EBSHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceEC2
}

func (h *EBSHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	volumeType := aws.GetStringAttr(attrs, "type")
	if volumeType == "" {
		volumeType = aws.VolumeTypeGP2
	}

	apiName := ebsVolumeAPIName[volumeType]
	if apiName == "" {
		apiName = "gp2"
	}

	lb := &aws.LookupBuilder{Service: pricing.ServiceEC2, ProductFamily: "Storage"}
	return lb.Build(region, map[string]string{
		"volumeApiName": apiName,
	}), nil
}

func (h *EBSHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	d := map[string]string{}
	if v := aws.GetStringAttr(attrs, "type"); v != "" {
		d["volume_type"] = v
	}
	if v := aws.GetFloatAttr(attrs, "size"); v > 0 {
		d["size_gb"] = fmt.Sprintf("%.0f", v)
	}
	if v := aws.GetFloatAttr(attrs, "iops"); v > 0 {
		d["iops"] = fmt.Sprintf("%.0f", v)
	}
	if v := aws.GetFloatAttr(attrs, "throughput"); v > 0 {
		d["throughput_mbps"] = fmt.Sprintf("%.0f", v)
	}
	return d
}

// CalculateCost calculates EBS cost using the price index for
// secondary lookups (IOPS, throughput). Falls back to hardcoded values
// when the index is nil or the lookup fails.
func (h *EBSHandler) CalculateCost(price *pricing.Price, index *pricing.PriceIndex, region string, attrs map[string]any) (hourly, monthly float64) {
	size := aws.GetFloatAttr(attrs, "size")
	if size == 0 {
		size = 8
	}

	// Storage cost (per GB-month from primary API lookup)
	monthly = price.OnDemandUSD * size

	volumeType := aws.GetStringAttr(attrs, "type")

	// IOPS cost for io1/io2
	if volumeType == aws.VolumeTypeIO1 || volumeType == aws.VolumeTypeIO2 {
		iops := aws.GetFloatAttr(attrs, "iops")
		if iops > 0 {
			suffix := "piops"
			fallback := FallbackIO1IOPSCostPerMonth
			if volumeType == aws.VolumeTypeIO2 {
				suffix = "io2"
				fallback = FallbackIO2IOPSCostPerMonth
			}
			iopsCost, ok := lookupEBSPrice(index, region, "System Operation", "EBS:VolumeP-IOPS."+suffix)
			if !ok {
				iopsCost = fallback
			}
			monthly += iops * iopsCost
		}
	}

	// gp3: IOPS cost above free tier (3000 free)
	if volumeType == aws.VolumeTypeGP3 {
		iops := aws.GetFloatAttr(attrs, "iops")
		if iops > DefaultGP3FreeIOPS {
			iopsCost, ok := lookupEBSPrice(index, region, "System Operation", "EBS:VolumeP-IOPS.gp3")
			if !ok {
				iopsCost = FallbackGP3IOPSCostPerMonth
			}
			monthly += (iops - DefaultGP3FreeIOPS) * iopsCost
		}

		// gp3: throughput cost above free tier (125 MBps free)
		throughput := aws.GetFloatAttr(attrs, "throughput")
		if throughput > DefaultGP3FreeThroughputMBps {
			tpCost, ok := lookupEBSPrice(index, region, "Provisioned Throughput", "EBS:VolumeP-Throughput.gp3")
			if !ok {
				tpCost = FallbackGP3ThroughputCostPerMB
			}
			monthly += (throughput - DefaultGP3FreeThroughputMBps) * tpCost
		}
	}

	hourly = monthly / aws.HoursPerMonth
	return hourly, monthly
}

// lookupEBSPrice finds a price in the index by product family and usagetype.
// Tries with regional prefix first, then without (us-east-1 uses unprefixed usagetypes).
func lookupEBSPrice(index *pricing.PriceIndex, region, productFamily, usageSuffix string) (float64, bool) {
	if index == nil {
		return 0, false
	}

	location := aws.ResolveRegionName(region)
	prefix := aws.ResolveUsagePrefix(region)

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
