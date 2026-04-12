package ec2

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

// EBS pricing fallbacks (used when API lookup unavailable).
const (
	FallbackIO1IOPSCostPerMonth    = 0.065
	FallbackIO2IOPSCostPerMonth    = 0.065
	FallbackGP3IOPSCostPerMonth    = 0.006
	FallbackGP3ThroughputCostPerMB = 0.040
)

// GP3 free tier thresholds (AWS always includes these without extra charge).
const (
	DefaultGP3FreeIOPS           = 3000
	DefaultGP3FreeThroughputMBps = 125
)

type ebsVolumeAttrs struct {
	VolumeType    string
	VolumeTypeSet bool
	SizeGB        float64
	SizeGBSet     bool
	IOPS          float64
	Throughput    float64
}

func parseEBSVolumeAttrs(attrs map[string]any) ebsVolumeAttrs {
	volumeType := handler.GetStringAttr(attrs, "type")
	sizeGB := handler.GetFloatAttr(attrs, "size")
	parsed := ebsVolumeAttrs{
		VolumeType:    volumeType,
		VolumeTypeSet: volumeType != "",
		SizeGB:        sizeGB,
		SizeGBSet:     sizeGB != 0,
		IOPS:          handler.GetFloatAttr(attrs, "iops"),
		Throughput:    handler.GetFloatAttr(attrs, "throughput"),
	}
	if parsed.VolumeType == "" {
		parsed.VolumeType = awskit.VolumeTypeGP2
	}
	if parsed.SizeGB == 0 {
		parsed.SizeGB = defaultRootVolumeGB
	}
	return parsed
}

// EBSSpec declares aws_ebs_volume cost estimation.
func EBSSpec(deps awskit.RuntimeDeps) resourcespec.ResourceSpec {
	ec2ServiceID := deps.RuntimeOrDefault().MustService(awskit.ServiceKeyEC2)

	return resourcespec.ResourceSpec{
		Type:     handler.ResourceType(awskit.ResourceEBSVolume),
		Category: handler.CostCategoryStandard,
		Lookup: &resourcespec.LookupSpec{
			BuildFunc: func(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
				parsed := parseEBSVolumeAttrs(attrs)
				lb := &awskit.PriceLookupSpec{Service: ec2ServiceID, ProductFamily: "Storage"}
				return lb.Lookup(region, map[string]string{
					"volumeApiName": parsed.VolumeType,
				}), nil
			},
		},
		Describe: &resourcespec.DescribeSpec{
			BuildFunc: func(_ *pricing.Price, attrs map[string]any) map[string]string {
				parsed := parseEBSVolumeAttrs(attrs)
				volumeType := ""
				if parsed.VolumeTypeSet {
					volumeType = parsed.VolumeType
				}
				sizeGB := 0.0
				if parsed.SizeGBSet {
					sizeGB = parsed.SizeGB
				}
				return awskit.NewDescribeBuilder().
					String("volume_type", volumeType).
					Float("size_gb", sizeGB, "%.0f").
					Float("iops", parsed.IOPS, "%.0f").
					Float("throughput_mbps", parsed.Throughput, "%.0f").
					Map()
			},
		},
		Standard: &resourcespec.StandardPricingSpec{
			CostFunc: func(price *pricing.Price, index *pricing.PriceIndex, region string, attrs map[string]any) (hourly, monthly float64) {
				if price == nil {
					return 0, 0
				}
				parsed := parseEBSVolumeAttrs(attrs)
				monthly = price.OnDemandUSD * parsed.SizeGB

				if parsed.VolumeType == awskit.VolumeTypeIO1 || parsed.VolumeType == awskit.VolumeTypeIO2 {
					if parsed.IOPS > 0 {
						suffix := "piops"
						fallback := FallbackIO1IOPSCostPerMonth
						if parsed.VolumeType == awskit.VolumeTypeIO2 {
							suffix = "io2"
							fallback = FallbackIO2IOPSCostPerMonth
						}
						iopsCost, ok := lookupEBSPrice(deps.RuntimeOrDefault(), index, region, "System Operation", "EBS:VolumeP-IOPS."+suffix)
						if !ok {
							iopsCost = fallback
						}
						monthly += parsed.IOPS * iopsCost
					}
				}

				if parsed.VolumeType == awskit.VolumeTypeGP3 {
					if parsed.IOPS > DefaultGP3FreeIOPS {
						iopsCost, ok := lookupEBSPrice(deps.RuntimeOrDefault(), index, region, "System Operation", "EBS:VolumeP-IOPS.gp3")
						if !ok {
							iopsCost = FallbackGP3IOPSCostPerMonth
						}
						monthly += (parsed.IOPS - DefaultGP3FreeIOPS) * iopsCost
					}
					if parsed.Throughput > DefaultGP3FreeThroughputMBps {
						tpCost, ok := lookupEBSPrice(deps.RuntimeOrDefault(), index, region, "Provisioned Throughput", "EBS:VolumeP-Throughput.gp3")
						if !ok {
							tpCost = FallbackGP3ThroughputCostPerMB
						}
						monthly += (parsed.Throughput - DefaultGP3FreeThroughputMBps) * tpCost
					}
				}

				return monthly / handler.HoursPerMonth, monthly
			},
		},
	}
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
