package ec2

import (
	"errors"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

// defaultRootVolumeGB is the default root volume size for EC2 instances.
const defaultRootVolumeGB = 8

type instanceAttrs struct {
	InstanceType string
	Tenancy      string
	RootVolume   ebsVolumeAttrs
}

func parseInstanceAttrs(attrs map[string]any) instanceAttrs {
	parsed := instanceAttrs{
		InstanceType: costutil.GetStringAttr(attrs, "instance_type"),
		Tenancy:      costutil.GetStringAttr(attrs, "tenancy"),
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
		VolumeType: costutil.GetStringAttr(attrs, "volume_type"),
		SizeGB:     costutil.GetFloatAttr(attrs, "volume_size"),
		IOPS:       costutil.GetFloatAttr(attrs, "iops"),
		Throughput: costutil.GetFloatAttr(attrs, "throughput"),
	}
	if parsed.VolumeType == "" {
		parsed.VolumeType = awskit.VolumeTypeGP2
	}
	if parsed.SizeGB == 0 {
		parsed.SizeGB = defaultRootVolumeGB
	}
	return parsed
}

// InstanceSpec declares aws_instance cost estimation.
func InstanceSpec(deps awskit.RuntimeDeps) resourcespec.TypedSpec[instanceAttrs] {
	return resourcespec.TypedSpec[instanceAttrs]{
		Type:     resourcedef.ResourceType(awskit.ResourceInstance),
		Category: resourcedef.CostCategoryStandard,
		Parse:    parseInstanceAttrs,
		Lookup: &resourcespec.TypedLookupSpec[instanceAttrs]{
			BuildFunc: func(region string, p instanceAttrs) (*pricing.PriceLookup, error) {
				if p.InstanceType == "" {
					return nil, errors.New("instance_type not found")
				}

				runtime := deps.RuntimeOrDefault()
				return runtime.
					NewLookupBuilder(awskit.ServiceKeyEC2, "Compute Instance").
					Attr("instanceType", p.InstanceType).
					Attr("operatingSystem", "Linux").
					Attr("preInstalledSw", "NA").
					Attr("capacitystatus", "Used").
					AttrMatch("tenancy", p.Tenancy, p.Tenancy, map[string]string{
						"":          "Shared",
						"default":   "Shared",
						"dedicated": "Dedicated",
						"host":      "Host",
					}).
					Build(region), nil
			},
		},
		Describe: &resourcespec.TypedDescribeSpec[instanceAttrs]{
			BuildFunc: func(_ *pricing.Price, p instanceAttrs) map[string]string {
				return awskit.NewDescribeBuilder().
					String("instance_type", p.InstanceType).
					StringIf(p.Tenancy != "" && p.Tenancy != "default", "tenancy", p.Tenancy).
					Map()
			},
		},
		Standard: &resourcespec.TypedStandardPricingSpec[instanceAttrs]{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ instanceAttrs) (hourly, monthly float64) {
				return awskit.NewCostBuilder().Hourly().Calc(price, nil, "")
			},
		},
		Subresources: &resourcespec.TypedSubresourceSpec[instanceAttrs]{
			BuildFunc: func(p instanceAttrs) []resourcedef.SubResource {
				root := p.RootVolume
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

				return []resourcedef.SubResource{{
					Suffix: "/root_volume",
					Type:   resourcedef.ResourceType(awskit.ResourceEBSVolume),
					Attrs:  ebsAttrs,
				}}
			},
		},
	}
}

// getRootBlockDevice extracts root_block_device from instance attributes.
func getRootBlockDevice(attrs map[string]any) map[string]any {
	return costutil.GetFirstObjectAttr(attrs, "root_block_device")
}
