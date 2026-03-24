package elb

import (
	aws "github.com/edelwud/terraci/plugins/cost/engine/aws"
	"github.com/edelwud/terraci/plugins/cost/engine/pricing"
)

// LB pricing constants
const (
	UsageType       = "LoadBalancerUsage"
	TypeApplication = "application"
	TypeNetwork     = "network"
	TypeGateway     = "gateway"

	// Default hourly costs
	DefaultALBHourlyCost  = 0.0225
	DefaultNLBHourlyCost  = 0.0225
	DefaultGWLBHourlyCost = 0.0125
)

// ALBHandler handles aws_lb (ALB/NLB) cost estimation
type ALBHandler struct{}

func (h *ALBHandler) Category() aws.CostCategory { return aws.CostCategoryStandard }

func (h *ALBHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceEC2
}

func (h *ALBHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	lbType := aws.GetStringAttr(attrs, "load_balancer_type")
	if lbType == "" {
		lbType = TypeApplication // Default to ALB
	}

	// Product family differs by LB type
	var productFamily string
	switch lbType {
	case TypeNetwork:
		productFamily = "Load Balancer-Network"
	case TypeGateway:
		productFamily = "Load Balancer-Gateway"
	default:
		productFamily = "Load Balancer-Application"
	}

	lb := &aws.LookupBuilder{Service: pricing.ServiceEC2, ProductFamily: productFamily}
	return lb.Build(region, map[string]string{
		"usagetype": region + "-" + UsageType,
	}), nil
}

func (h *ALBHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	d := map[string]string{}
	lbType := aws.GetStringAttr(attrs, "load_balancer_type")
	if lbType == "" {
		lbType = TypeApplication
	}
	d["type"] = lbType
	return d
}

func (h *ALBHandler) CalculateCost(price *pricing.Price, _ *pricing.PriceIndex, _ string, attrs map[string]any) (hourly, monthly float64) {
	rate := price.OnDemandUSD
	if rate == 0 {
		// Default pricing if lookup fails
		lbType := aws.GetStringAttr(attrs, "load_balancer_type")
		switch lbType {
		case TypeNetwork:
			rate = DefaultNLBHourlyCost
		case TypeGateway:
			rate = DefaultGWLBHourlyCost
		default:
			rate = DefaultALBHourlyCost
		}
	}
	return aws.HourlyCost(rate)
}
