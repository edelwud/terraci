package aws

import (
	"fmt"

	"github.com/edelwud/terraci/internal/cost/pricing"
)

// EKS pricing constants
const (
	DefaultEKSClusterHourlyCost = 0.10
	DefaultEKSInstanceType      = "t3.medium"
)

// EKSClusterHandler handles aws_eks_cluster cost estimation
type EKSClusterHandler struct{}

func (h *EKSClusterHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceEKS
}

func (h *EKSClusterHandler) BuildLookup(region string, _ map[string]interface{}) (*pricing.PriceLookup, error) {
	regionName := pricing.RegionMapping[region]
	if regionName == "" {
		regionName = region
	}

	return &pricing.PriceLookup{
		ServiceCode:   pricing.ServiceEKS,
		Region:        region,
		ProductFamily: "Compute",
		Attributes: map[string]string{
			"location":  regionName,
			"usagetype": region + "-AmazonEKS-Hours:perCluster",
		},
	}, nil
}

func (h *EKSClusterHandler) CalculateCost(price *pricing.Price, _ map[string]interface{}) (hourly, monthly float64) {
	hourly = price.OnDemandUSD
	if hourly == 0 {
		hourly = DefaultEKSClusterHourlyCost
	}
	monthly = hourly * HoursPerMonth
	return hourly, monthly
}

// EKSNodeGroupHandler handles aws_eks_node_group cost estimation
type EKSNodeGroupHandler struct{}

func (h *EKSNodeGroupHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceEC2
}

func (h *EKSNodeGroupHandler) BuildLookup(region string, attrs map[string]interface{}) (*pricing.PriceLookup, error) {
	// Get instance types from node group
	var instanceType string

	// Instance types can be in different locations depending on terraform version
	if instanceTypes, ok := attrs["instance_types"].([]interface{}); ok && len(instanceTypes) > 0 {
		if t, ok := instanceTypes[0].(string); ok {
			instanceType = t
		}
	}

	if instanceType == "" {
		instanceType = DefaultEKSInstanceType
	}

	regionName := pricing.RegionMapping[region]
	if regionName == "" {
		regionName = region
	}

	return &pricing.PriceLookup{
		ServiceCode:   pricing.ServiceEC2,
		Region:        region,
		ProductFamily: "Compute Instance",
		Attributes: map[string]string{
			"instanceType":    instanceType,
			"location":        regionName,
			"tenancy":         "Shared",
			"operatingSystem": "Linux",
			"preInstalledSw":  "NA",
			"capacitystatus":  "Used",
		},
	}, nil
}

func (h *EKSNodeGroupHandler) CalculateCost(price *pricing.Price, attrs map[string]interface{}) (hourly, monthly float64) {
	// Determine node count from scaling_config
	desiredSize := 1

	if scalingConfig, ok := attrs["scaling_config"].([]interface{}); ok && len(scalingConfig) > 0 {
		if cfg, ok := scalingConfig[0].(map[string]interface{}); ok {
			if d := getIntAttr(cfg, "desired_size"); d > 0 {
				desiredSize = d
			}
		}
	}

	hourly = price.OnDemandUSD * float64(desiredSize)
	monthly = hourly * HoursPerMonth
	return hourly, monthly
}

// ECSClusterHandler handles aws_ecs_cluster cost estimation
// Note: ECS cluster itself is free, cost comes from tasks/services
type ECSClusterHandler struct{}

func (h *ECSClusterHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceECS
}

func (h *ECSClusterHandler) BuildLookup(_ string, _ map[string]interface{}) (*pricing.PriceLookup, error) {
	// ECS cluster has no direct cost, return nil
	return nil, fmt.Errorf("ECS cluster has no direct cost")
}

func (h *ECSClusterHandler) CalculateCost(_ *pricing.Price, _ map[string]interface{}) (hourly, monthly float64) {
	// ECS cluster is free
	return 0, 0
}
