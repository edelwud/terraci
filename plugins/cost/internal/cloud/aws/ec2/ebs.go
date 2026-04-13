package ec2

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
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
	volumeType := costutil.GetStringAttr(attrs, "type")
	sizeGB := costutil.GetFloatAttr(attrs, "size")
	parsed := ebsVolumeAttrs{
		VolumeType:    volumeType,
		VolumeTypeSet: volumeType != "",
		SizeGB:        sizeGB,
		SizeGBSet:     sizeGB != 0,
		IOPS:          costutil.GetFloatAttr(attrs, "iops"),
		Throughput:    costutil.GetFloatAttr(attrs, "throughput"),
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
func EBSSpec(deps awskit.RuntimeDeps) resourcespec.TypedSpec[ebsVolumeAttrs] {
	return resourcespec.TypedSpec[ebsVolumeAttrs]{
		Type:     resourcedef.ResourceType(awskit.ResourceEBSVolume),
		Category: resourcedef.CostCategoryStandard,
		Parse:    parseEBSVolumeAttrs,
		Lookup: &resourcespec.TypedLookupSpec[ebsVolumeAttrs]{
			BuildFunc: func(region string, p ebsVolumeAttrs) (*pricing.PriceLookup, error) {
				return deps.RuntimeOrDefault().
					NewLookupBuilder(awskit.ServiceKeyEC2, "Storage").
					Attr("volumeApiName", p.VolumeType).
					Build(region), nil
			},
		},
		Describe: &resourcespec.TypedDescribeSpec[ebsVolumeAttrs]{
			BuildFunc: func(_ *pricing.Price, p ebsVolumeAttrs) map[string]string {
				return awskit.NewDescribeBuilder().
					StringIf(p.VolumeTypeSet, "volume_type", p.VolumeType).
					FloatIf(p.SizeGBSet, "size_gb", p.SizeGB, "%.0f").
					Float("iops", p.IOPS, "%.0f").
					Float("throughput_mbps", p.Throughput, "%.0f").
					Map()
			},
		},
		Standard: &resourcespec.TypedStandardPricingSpec[ebsVolumeAttrs]{
			CostFunc: func(price *pricing.Price, index *pricing.PriceIndex, region string, p ebsVolumeAttrs) (hourly, monthly float64) {
				rt := deps.RuntimeOrDefault()
				return awskit.NewCostBuilder().
					PerUnit(p.SizeGB).
					Match(p.VolumeType, nil, map[string][]awskit.Charge{
						awskit.VolumeTypeIO1: {
							awskit.NewCharge(p.IOPS).
								Rate(awskit.IndexRate(rt, "System Operation", "EBS:VolumeP-IOPS.piops")).
								Fallback(FallbackIO1IOPSCostPerMonth),
						},
						awskit.VolumeTypeIO2: {
							awskit.NewCharge(p.IOPS).
								Rate(awskit.IndexRate(rt, "System Operation", "EBS:VolumeP-IOPS.io2")).
								Fallback(FallbackIO2IOPSCostPerMonth),
						},
						awskit.VolumeTypeGP3: {
							awskit.NewCharge(p.IOPS).
								FreeTier(DefaultGP3FreeIOPS).
								Rate(awskit.IndexRate(rt, "System Operation", "EBS:VolumeP-IOPS.gp3")).
								Fallback(FallbackGP3IOPSCostPerMonth),
							awskit.NewCharge(p.Throughput).
								FreeTier(DefaultGP3FreeThroughputMBps).
								Rate(awskit.IndexRate(rt, "Provisioned Throughput", "EBS:VolumeP-Throughput.gp3")).
								Fallback(FallbackGP3ThroughputCostPerMB),
						},
					}).
					Calc(price, index, region)
			},
		},
	}
}
