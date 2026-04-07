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
	ec2ServiceID pricing.ServiceID
}

// NewEBSHandler creates an EBSHandler with a pre-resolved EC2 service ID.
func NewEBSHandler(deps awskit.RuntimeDeps) *EBSHandler {
	return &EBSHandler{RuntimeDeps: deps, ec2ServiceID: deps.RuntimeOrDefault().MustService(awskit.ServiceKeyEC2)}
}

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

func (h *EBSHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *EBSHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	parsed := parseEBSVolumeAttrs(attrs)
	apiName := ebsVolumeAPIName[parsed.VolumeType]
	if apiName == "" {
		apiName = "gp2"
	}

	lb := &awskit.PriceLookupSpec{Service: h.ec2ServiceID, ProductFamily: "Storage"}
	return lb.Lookup(region, map[string]string{
		"volumeApiName": apiName,
	}), nil
}

func (h *EBSHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
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
}

// CalculateCost calculates EBS cost using the price index for
// secondary lookups (IOPS, throughput). Falls back to hardcoded values
// when the index is nil or the lookup fails.
func (h *EBSHandler) CalculateCost(price *pricing.Price, index *pricing.PriceIndex, region string, attrs map[string]any) (hourly, monthly float64) {
	parsed := parseEBSVolumeAttrs(attrs)

	// Storage cost (per GB-month from primary API lookup)
	monthly = price.OnDemandUSD * parsed.SizeGB

	// IOPS cost for io1/io2
	if parsed.VolumeType == awskit.VolumeTypeIO1 || parsed.VolumeType == awskit.VolumeTypeIO2 {
		if parsed.IOPS > 0 {
			suffix := "piops"
			fallback := FallbackIO1IOPSCostPerMonth
			if parsed.VolumeType == awskit.VolumeTypeIO2 {
				suffix = "io2"
				fallback = FallbackIO2IOPSCostPerMonth
			}
			iopsCost, ok := lookupEBSPrice(h.RuntimeOrDefault(), index, region, "System Operation", "EBS:VolumeP-IOPS."+suffix)
			if !ok {
				iopsCost = fallback
			}
			monthly += parsed.IOPS * iopsCost
		}
	}

	// gp3: IOPS cost above free tier (3000 free)
	if parsed.VolumeType == awskit.VolumeTypeGP3 {
		if parsed.IOPS > DefaultGP3FreeIOPS {
			iopsCost, ok := lookupEBSPrice(h.RuntimeOrDefault(), index, region, "System Operation", "EBS:VolumeP-IOPS.gp3")
			if !ok {
				iopsCost = FallbackGP3IOPSCostPerMonth
			}
			monthly += (parsed.IOPS - DefaultGP3FreeIOPS) * iopsCost
		}

		// gp3: throughput cost above free tier (125 MBps free)
		if parsed.Throughput > DefaultGP3FreeThroughputMBps {
			tpCost, ok := lookupEBSPrice(h.RuntimeOrDefault(), index, region, "Provisioned Throughput", "EBS:VolumeP-Throughput.gp3")
			if !ok {
				tpCost = FallbackGP3ThroughputCostPerMB
			}
			monthly += (parsed.Throughput - DefaultGP3FreeThroughputMBps) * tpCost
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
