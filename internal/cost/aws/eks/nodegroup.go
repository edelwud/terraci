package eks

import (
	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

const (
	DefaultInstanceType = "t3.medium"
)

// NodeGroupHandler handles aws_eks_node_group cost estimation
type NodeGroupHandler struct{}

func (h *NodeGroupHandler) Category() aws.CostCategory { return aws.CostCategoryStandard }

func (h *NodeGroupHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceEC2
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

	lb := &aws.LookupBuilder{Service: pricing.ServiceEC2, ProductFamily: "Compute Instance"}
	return lb.Build(region, map[string]string{
		"instanceType":    instanceType,
		"tenancy":         "Shared",
		"operatingSystem": "Linux",
		"preInstalledSw":  "NA",
		"capacitystatus":  "Used",
	}), nil
}

func (h *NodeGroupHandler) CalculateCost(price *pricing.Price, attrs map[string]any) (hourly, monthly float64) {
	// Determine node count from scaling_config
	desiredSize := 1

	if scalingConfig, ok := attrs["scaling_config"].([]any); ok && len(scalingConfig) > 0 {
		if cfg, ok := scalingConfig[0].(map[string]any); ok {
			if d := aws.GetIntAttr(cfg, "desired_size"); d > 0 {
				desiredSize = d
			}
		}
	}

	return aws.ScaledHourlyCost(price.OnDemandUSD, desiredSize)
}
