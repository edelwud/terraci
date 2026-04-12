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
func InstanceSpec(deps awskit.RuntimeDeps) resourcespec.ResourceSpec {
	return resourcespec.ResourceSpec{
		Type:     resourcedef.ResourceType(awskit.ResourceInstance),
		Category: resourcedef.CostCategoryStandard,
		Lookup: &resourcespec.LookupSpec{
			BuildFunc: func(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
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

				return deps.RuntimeOrDefault().StandardLookupSpec(
					awskit.ServiceKeyEC2,
					"Compute Instance",
					func(_ string, _ map[string]any) (map[string]string, error) {
						return map[string]string{
							"instanceType":    parsed.InstanceType,
							"tenancy":         tenancy,
							"operatingSystem": "Linux",
							"preInstalledSw":  "NA",
							"capacitystatus":  "Used",
						}, nil
					},
				).Build(region, attrs)
			},
		},
		Describe: &resourcespec.DescribeSpec{
			BuildFunc: func(_ *pricing.Price, attrs map[string]any) map[string]string {
				parsed := parseInstanceAttrs(attrs)
				desc := awskit.NewDescribeBuilder().
					String("instance_type", parsed.InstanceType)
				if parsed.Tenancy != "" && parsed.Tenancy != "default" {
					desc.String("tenancy", parsed.Tenancy)
				}
				return desc.Map()
			},
		},
		Standard: &resourcespec.StandardPricingSpec{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
				return costutil.HourlyCost(price.OnDemandUSD)
			},
		},
		Subresources: &resourcespec.SubresourceSpec{
			BuildFunc: func(attrs map[string]any) []resourcedef.SubResource {
				root := parseInstanceAttrs(attrs).RootVolume
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
