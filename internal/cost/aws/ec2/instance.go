package ec2

import (
	"fmt"

	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

// defaultRootVolumeGB is the default root volume size for EC2 instances.
const defaultRootVolumeGB = 8

// InstanceHandler handles aws_instance cost estimation.
type InstanceHandler struct{}

func (h *InstanceHandler) Category() aws.CostCategory { return aws.CostCategoryStandard }

func (h *InstanceHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceEC2
}

func (h *InstanceHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	instanceType := aws.GetStringAttr(attrs, "instance_type")
	if instanceType == "" {
		return nil, fmt.Errorf("instance_type not found")
	}

	tenancy := aws.GetStringAttr(attrs, "tenancy")
	switch tenancy {
	case "", "default":
		tenancy = "Shared"
	case "dedicated":
		tenancy = "Dedicated"
	case "host":
		tenancy = "Host"
	}

	operatingSystem := "Linux"
	if ami := aws.GetStringAttr(attrs, "ami"); ami != "" {
		operatingSystem = "Linux"
	}

	lb := &aws.LookupBuilder{Service: pricing.ServiceEC2, ProductFamily: "Compute Instance"}
	return lb.Build(region, map[string]string{
		"instanceType":    instanceType,
		"tenancy":         tenancy,
		"operatingSystem": operatingSystem,
		"preInstalledSw":  "NA",
		"capacitystatus":  "Used",
	}), nil
}

func (h *InstanceHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	d := map[string]string{}
	if v := aws.GetStringAttr(attrs, "instance_type"); v != "" {
		d["instance_type"] = v
	}
	if v := aws.GetStringAttr(attrs, "tenancy"); v != "" && v != "default" {
		d["tenancy"] = v
	}
	return d
}

func (h *InstanceHandler) CalculateCost(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
	return aws.HourlyCost(price.OnDemandUSD)
}

// SubResources synthesizes sub-resources from inline attributes.
// root_block_device is dispatched to EBSHandler as a virtual aws_ebs_volume.
func (h *InstanceHandler) SubResources(attrs map[string]any) []aws.SubResource {
	root := getRootBlockDevice(attrs)
	if root == nil {
		// Default root volume: 8 GB gp2
		root = map[string]any{
			"volume_type": aws.VolumeTypeGP2,
			"volume_size": float64(defaultRootVolumeGB),
		}
	}

	// Translate root_block_device attrs → aws_ebs_volume attrs
	ebsAttrs := map[string]any{
		"type": aws.GetStringAttr(root, "volume_type"),
		"size": aws.GetFloatAttr(root, "volume_size"),
	}
	if ebsAttrs["type"] == "" {
		ebsAttrs["type"] = aws.VolumeTypeGP2
	}
	if ebsAttrs["size"] == float64(0) {
		ebsAttrs["size"] = float64(defaultRootVolumeGB)
	}
	if iops := aws.GetFloatAttr(root, "iops"); iops > 0 {
		ebsAttrs["iops"] = iops
	}
	if tp := aws.GetFloatAttr(root, "throughput"); tp > 0 {
		ebsAttrs["throughput"] = tp
	}

	return []aws.SubResource{
		{
			Suffix: "/root_volume",
			Type:   "aws_ebs_volume",
			Attrs:  ebsAttrs,
		},
	}
}

// getRootBlockDevice extracts root_block_device from instance attributes.
func getRootBlockDevice(attrs map[string]any) map[string]any {
	rbd, ok := attrs["root_block_device"]
	if !ok {
		return nil
	}
	list, ok := rbd.([]any)
	if !ok || len(list) == 0 {
		return nil
	}
	m, ok := list[0].(map[string]any)
	if !ok {
		return nil
	}
	return m
}
