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
				return deps.RuntimeOrDefault().
					NewLookupBuilder(awskit.ServiceKeyEC2, "Compute Instance").
					Attr("instanceType", p.InstanceType).
					Attr("tenancy", "Shared").
					Attr("operatingSystem", "Linux").
					Attr("preInstalledSw", "NA").
					Attr("capacitystatus", "Used").
					Build(region), nil
			},
		},
		Describe: &resourcespec.TypedDescribeSpec[nodeGroupAttrs]{
			BuildFunc: func(_ *pricing.Price, p nodeGroupAttrs) map[string]string {
				return awskit.NewDescribeBuilder().
					StringIf(p.InstanceTypeSet, "instance_type", p.InstanceType).
					IntIf(p.DesiredSizeSet, "desired_size", p.DesiredSize).
					Map()
			},
		},
		Standard: &resourcespec.TypedStandardPricingSpec[nodeGroupAttrs]{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, p nodeGroupAttrs) (hourly, monthly float64) {
				return awskit.NewCostBuilder().Hourly().Scale(float64(p.DesiredSize)).Calc(price, nil, "")
			},
		},
	}
}
