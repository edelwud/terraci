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
func NodeGroupSpec(deps awskit.RuntimeDeps) resourcespec.TypedSpec[nodeGroupAttrs] {
	return resourcespec.TypedSpec[nodeGroupAttrs]{
		Type:     resourcedef.ResourceType(awskit.ResourceEKSNodeGroup),
		Category: resourcedef.CostCategoryStandard,
		Parse:    parseNodeGroupAttrs,
		Lookup: &resourcespec.TypedLookupSpec[nodeGroupAttrs]{
			BuildFunc: func(region string, p nodeGroupAttrs) (*pricing.PriceLookup, error) {
				return deps.RuntimeOrDefault().StandardLookupSpec(
					awskit.ServiceKeyEC2,
					"Compute Instance",
					func(_ string, _ map[string]any) (map[string]string, error) {
						return map[string]string{
							"instanceType":    p.InstanceType,
							"tenancy":         "Shared",
							"operatingSystem": "Linux",
							"preInstalledSw":  "NA",
							"capacitystatus":  "Used",
						}, nil
					},
				).Build(region, nil)
			},
		},
		Describe: &resourcespec.TypedDescribeSpec[nodeGroupAttrs]{
			BuildFunc: func(_ *pricing.Price, p nodeGroupAttrs) map[string]string {
				desc := awskit.NewDescribeBuilder()
				if p.InstanceTypeSet {
					desc.String("instance_type", p.InstanceType)
				}
				if p.DesiredSizeSet {
					desc.Int("desired_size", p.DesiredSize)
				}
				return desc.Map()
			},
		},
		Standard: &resourcespec.TypedStandardPricingSpec[nodeGroupAttrs]{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, p nodeGroupAttrs) (hourly, monthly float64) {
				if price == nil {
					return 0, 0
				}
				return costutil.ScaledHourlyCost(price.OnDemandUSD, p.DesiredSize)
			},
		},
	}
}
