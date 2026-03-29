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

func (h *NodeGroupHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *NodeGroupHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	desc := awskit.NewDescribeBuilder()

	var instanceType string
	if instanceTypes, ok := attrs["instance_types"].([]any); ok && len(instanceTypes) > 0 {
		if t, ok := instanceTypes[0].(string); ok {
			instanceType = t
		}
	}
	if instanceType != "" {
		desc.String("instance_type", instanceType)
	}

	if scalingConfig, ok := attrs["scaling_config"].([]any); ok && len(scalingConfig) > 0 {
		if cfg, ok := scalingConfig[0].(map[string]any); ok {
			if d := handler.GetIntAttr(cfg, "desired_size"); d > 0 {
				desc.Int("desired_size", d)
			}
		}
	}
	return desc.Map()
}

func (h *NodeGroupHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	// Get instance types from node group
	var instanceType string

	// Instance types can be in different locations depending on terraform version
	if instanceTypes, ok := attrs["instance_types"].([]any); ok && len(instanceTypes) > 0 {
		if t, ok := instanceTypes[0].(string); ok {
			instanceType = t
		}
	}

	if instanceType == "" {
		instanceType = DefaultInstanceType
	}

	return h.RuntimeOrDefault().StandardLookupSpec(
		awskit.ServiceKeyEC2,
		"Compute Instance",
		func(_ string, _ map[string]any) (map[string]string, error) {
			return map[string]string{
				"instanceType":    instanceType,
				"tenancy":         "Shared",
				"operatingSystem": "Linux",
				"preInstalledSw":  "NA",
				"capacitystatus":  "Used",
			}, nil
		},
	).Build(region, attrs)
}

func (h *NodeGroupHandler) CalculateCost(price *pricing.Price, _ *pricing.PriceIndex, _ string, attrs map[string]any) (hourly, monthly float64) {
	// Determine node count from scaling_config
	desiredSize := 1

	if scalingConfig, ok := attrs["scaling_config"].([]any); ok && len(scalingConfig) > 0 {
		if cfg, ok := scalingConfig[0].(map[string]any); ok {
			if d := handler.GetIntAttr(cfg, "desired_size"); d > 0 {
				desiredSize = d
			}
		}
	}

	return handler.ScaledHourlyCost(price.OnDemandUSD, desiredSize)
}
