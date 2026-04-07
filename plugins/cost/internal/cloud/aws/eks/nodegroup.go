package eks

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

const (
	DefaultInstanceType = "t3.medium"
)

// NodeGroupHandler handles aws_eks_node_group cost estimation
type NodeGroupHandler struct {
	awskit.RuntimeDeps
}

type nodeGroupAttrs struct {
	InstanceType    string
	InstanceTypeSet bool
	DesiredSize     int
	DesiredSizeSet  bool
}

func parseNodeGroupAttrs(attrs map[string]any) nodeGroupAttrs {
	instanceType := ""
	if instanceTypes := handler.GetStringSliceAttr(attrs, "instance_types"); len(instanceTypes) > 0 {
		instanceType = instanceTypes[0]
	}
	instanceTypeSet := instanceType != ""
	if instanceType == "" {
		instanceType = DefaultInstanceType
	}

	desiredSize := 1
	desiredSizeSet := false
	if cfg := handler.GetFirstObjectAttr(attrs, "scaling_config"); cfg != nil {
		if d := handler.GetIntAttr(cfg, "desired_size"); d > 0 {
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

func (h *NodeGroupHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *NodeGroupHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	parsed := parseNodeGroupAttrs(attrs)
	desc := awskit.NewDescribeBuilder()

	if parsed.InstanceTypeSet {
		desc.String("instance_type", parsed.InstanceType)
	}

	if parsed.DesiredSizeSet {
		desc.Int("desired_size", parsed.DesiredSize)
	}
	return desc.Map()
}

func (h *NodeGroupHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	parsed := parseNodeGroupAttrs(attrs)

	return h.RuntimeOrDefault().StandardLookupSpec(
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
}

func (h *NodeGroupHandler) CalculateCost(price *pricing.Price, _ *pricing.PriceIndex, _ string, attrs map[string]any) (hourly, monthly float64) {
	parsed := parseNodeGroupAttrs(attrs)
	return handler.ScaledHourlyCost(price.OnDemandUSD, parsed.DesiredSize)
}
