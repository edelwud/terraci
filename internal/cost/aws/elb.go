package aws

import (
	"github.com/edelwud/terraci/internal/cost/pricing"
)

// LB pricing constants
const (
	LBUsageType       = "LoadBalancerUsage"
	LBTypeApplication = "application"
	LBTypeNetwork     = "network"
	LBTypeGateway     = "gateway"

	// Default hourly costs
	DefaultALBHourlyCost       = 0.0225
	DefaultNLBHourlyCost       = 0.0225
	DefaultGWLBHourlyCost      = 0.0125
	DefaultClassicLBHourlyCost = 0.025
)

// LBHandler handles aws_lb (ALB/NLB) cost estimation
type LBHandler struct{}

func (h *LBHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceEC2
}

func (h *LBHandler) BuildLookup(region string, attrs map[string]interface{}) (*pricing.PriceLookup, error) {
	lbType := getStringAttr(attrs, "load_balancer_type")
	if lbType == "" {
		lbType = LBTypeApplication // Default to ALB
	}

	regionName := pricing.RegionMapping[region]
	if regionName == "" {
		regionName = region
	}

	// Product family differs by LB type
	var productFamily string
	switch lbType {
	case LBTypeNetwork:
		productFamily = "Load Balancer-Network"
	case LBTypeGateway:
		productFamily = "Load Balancer-Gateway"
	default:
		productFamily = "Load Balancer-Application"
	}

	return &pricing.PriceLookup{
		ServiceCode:   pricing.ServiceEC2,
		Region:        region,
		ProductFamily: productFamily,
		Attributes: map[string]string{
			"location":  regionName,
			"usagetype": region + "-" + LBUsageType,
		},
	}, nil
}

func (h *LBHandler) CalculateCost(price *pricing.Price, attrs map[string]interface{}) (hourly, monthly float64) {
	hourly = price.OnDemandUSD
	if hourly == 0 {
		// Default pricing if lookup fails
		lbType := getStringAttr(attrs, "load_balancer_type")
		switch lbType {
		case LBTypeNetwork:
			hourly = DefaultNLBHourlyCost
		case LBTypeGateway:
			hourly = DefaultGWLBHourlyCost
		default:
			hourly = DefaultALBHourlyCost
		}
	}
	monthly = hourly * HoursPerMonth
	return hourly, monthly
}

// ClassicLBHandler handles aws_elb (Classic) cost estimation
type ClassicLBHandler struct{}

func (h *ClassicLBHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceELB
}

func (h *ClassicLBHandler) BuildLookup(region string, _ map[string]interface{}) (*pricing.PriceLookup, error) {
	regionName := pricing.RegionMapping[region]
	if regionName == "" {
		regionName = region
	}

	return &pricing.PriceLookup{
		ServiceCode:   pricing.ServiceELB,
		Region:        region,
		ProductFamily: "Load Balancer",
		Attributes: map[string]string{
			"location":  regionName,
			"usagetype": region + "-" + LBUsageType,
		},
	}, nil
}

func (h *ClassicLBHandler) CalculateCost(price *pricing.Price, _ map[string]interface{}) (hourly, monthly float64) {
	hourly = price.OnDemandUSD
	if hourly == 0 {
		hourly = DefaultClassicLBHourlyCost
	}
	monthly = hourly * HoursPerMonth
	return hourly, monthly
}
