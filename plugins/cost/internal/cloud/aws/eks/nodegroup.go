package eks

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

const (
	DefaultInstanceType = "t3.medium"
)

type nodeGroupAttrs struct {
	InstanceType    string
	InstanceTypeSet bool
	DesiredSize     int
	DesiredSizeSet  bool
}

func parseNodeGroupAttrs(attrs map[string]any) nodeGroupAttrs {
	instanceType := ""
	if instanceTypes := costutil.GetStringSliceAttr(attrs, "instance_types"); len(instanceTypes) > 0 {
		instanceType = instanceTypes[0]
	}
	instanceTypeSet := instanceType != ""
	if instanceType == "" {
		instanceType = DefaultInstanceType
	}

	desiredSize := 1
	desiredSizeSet := false
	if cfg := costutil.GetFirstObjectAttr(attrs, "scaling_config"); cfg != nil {
		if d := costutil.GetIntAttr(cfg, "desired_size"); d > 0 {
			desiredSize = d
			desiredSizeSet = true
		}
	}

	return nodeGroupAttrs{
		InstanceType:    instanceType,
		InstanceTypeSet: instanceTypeSet,
		DesiredSize:     desiredSize,
		DesiredSizeSet:  desiredSizeSet,
	}
}

// NodeGroupSpec declares aws_eks_node_group cost estimation.
func NodeGroupSpec(deps awskit.RuntimeDeps) resourcespec.ResourceSpec {
	return resourcespec.ResourceSpec{
		Type:     resourcedef.ResourceType(awskit.ResourceEKSNodeGroup),
		Category: resourcedef.CostCategoryStandard,
		Lookup: &resourcespec.LookupSpec{
			BuildFunc: func(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
				parsed := parseNodeGroupAttrs(attrs)
				return deps.RuntimeOrDefault().StandardLookupSpec(
					awskit.ServiceKeyEC2,
					"Compute Instance",
					func(_ string, _ map[string]any) (map[string]string, error) {
						return map[string]string{
							"instanceType":    parsed.InstanceType,
							"tenancy":         "Shared",
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
				parsed := parseNodeGroupAttrs(attrs)
				desc := awskit.NewDescribeBuilder()
				if parsed.InstanceTypeSet {
					desc.String("instance_type", parsed.InstanceType)
				}
				if parsed.DesiredSizeSet {
					desc.Int("desired_size", parsed.DesiredSize)
				}
				return desc.Map()
			},
		},
		Standard: &resourcespec.StandardPricingSpec{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, attrs map[string]any) (hourly, monthly float64) {
				if price == nil {
					return 0, 0
				}
				return costutil.ScaledHourlyCost(price.OnDemandUSD, parseNodeGroupAttrs(attrs).DesiredSize)
			},
		},
	}
}
