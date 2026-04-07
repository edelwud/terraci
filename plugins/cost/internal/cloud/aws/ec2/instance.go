package ec2

import (
	"errors"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// defaultRootVolumeGB is the default root volume size for EC2 instances.
const defaultRootVolumeGB = 8

// InstanceHandler handles aws_instance cost estimation.
type InstanceHandler struct {
	awskit.RuntimeDeps
}

type instanceAttrs struct {
	InstanceType string
	Tenancy      string
	RootVolume   ebsVolumeAttrs
}

func parseInstanceAttrs(attrs map[string]any) instanceAttrs {
	parsed := instanceAttrs{
		InstanceType: handler.GetStringAttr(attrs, "instance_type"),
		Tenancy:      handler.GetStringAttr(attrs, "tenancy"),
		RootVolume: ebsVolumeAttrs{
			VolumeType: awskit.VolumeTypeGP2,
			SizeGB:     defaultRootVolumeGB,
		},
	}
	if root := getRootBlockDevice(attrs); root != nil {
		parsed.RootVolume = parseRootVolumeAttrs(root)
	}
	return parsed
}

func parseRootVolumeAttrs(attrs map[string]any) ebsVolumeAttrs {
	parsed := ebsVolumeAttrs{
		VolumeType: handler.GetStringAttr(attrs, "volume_type"),
		SizeGB:     handler.GetFloatAttr(attrs, "volume_size"),
		IOPS:       handler.GetFloatAttr(attrs, "iops"),
		Throughput: handler.GetFloatAttr(attrs, "throughput"),
	}
	if parsed.VolumeType == "" {
		parsed.VolumeType = awskit.VolumeTypeGP2
	}
	if parsed.SizeGB == 0 {
		parsed.SizeGB = defaultRootVolumeGB
	}
	return parsed
}

func (h *InstanceHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *InstanceHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	parsed := parseInstanceAttrs(attrs)
	if parsed.InstanceType == "" {
		return nil, errors.New("instance_type not found")
	}

	tenancy := parsed.Tenancy
	switch tenancy {
	case "", "default":
		tenancy = "Shared"
	case "dedicated":
		tenancy = "Dedicated"
	case "host":
		tenancy = "Host"
	}

	operatingSystem := "Linux"

	return h.RuntimeOrDefault().StandardLookupSpec(
		awskit.ServiceKeyEC2,
		"Compute Instance",
		func(_ string, _ map[string]any) (map[string]string, error) {
			return map[string]string{
				"instanceType":    parsed.InstanceType,
				"tenancy":         tenancy,
				"operatingSystem": operatingSystem,
				"preInstalledSw":  "NA",
				"capacitystatus":  "Used",
			}, nil
		},
	).Build(region, attrs)
}

func (h *InstanceHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	parsed := parseInstanceAttrs(attrs)
	desc := awskit.NewDescribeBuilder().
		String("instance_type", parsed.InstanceType)
	if parsed.Tenancy != "" && parsed.Tenancy != "default" {
		desc.String("tenancy", parsed.Tenancy)
	}
	return desc.Map()
}

func (h *InstanceHandler) CalculateCost(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
	return handler.HourlyCost(price.OnDemandUSD)
}

// SubResources synthesizes sub-resources from inline attributes.
// root_block_device is dispatched to EBSHandler as a virtual aws_ebs_volume.
func (h *InstanceHandler) SubResources(attrs map[string]any) []handler.SubResource {
	root := parseInstanceAttrs(attrs).RootVolume

	// Translate root_block_device attrs → aws_ebs_volume attrs
	ebsAttrs := map[string]any{
		"type": root.VolumeType,
		"size": root.SizeGB,
	}
	if root.IOPS > 0 {
		ebsAttrs["iops"] = root.IOPS
	}
	if root.Throughput > 0 {
		ebsAttrs["throughput"] = root.Throughput
	}

	return []handler.SubResource{
		{
			Suffix: "/root_volume",
			Type:   handler.ResourceType(awskit.ResourceEBSVolume),
			Attrs:  ebsAttrs,
		},
	}
}

// getRootBlockDevice extracts root_block_device from instance attributes.
func getRootBlockDevice(attrs map[string]any) map[string]any {
	return handler.GetFirstObjectAttr(attrs, "root_block_device")
}
