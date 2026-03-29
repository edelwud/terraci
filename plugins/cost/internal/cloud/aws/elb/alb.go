package elb

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// LB pricing constants
const (
	UsageType        = "LoadBalancerUsage"
	TypeApplication  = "application"
	TypeNetwork      = "network"
	TypeGateway      = "gateway"
	ProductFamilyALB = "Load Balancer-Application"

	// Default hourly costs
	DefaultALBHourlyCost  = 0.0225
	DefaultNLBHourlyCost  = 0.0225
	DefaultGWLBHourlyCost = 0.0125
)

// ALBHandler handles aws_lb (ALB/NLB) cost estimation
type ALBHandler struct {
	awskit.RuntimeDeps
}

func (h *ALBHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *ALBHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	lbType := handler.GetStringAttr(attrs, "load_balancer_type")
	if lbType == "" {
		lbType = TypeApplication // Default to ALB
	}

	spec := h.RuntimeOrDefault().StandardLookupSpec(
		awskit.ServiceKeyEC2,
		"",
		func(region string, _ map[string]any) (map[string]string, error) {
			return map[string]string{
				"usagetype": region + "-" + UsageType,
			}, nil
		},
	)

	switch lbType {
	case TypeNetwork:
		spec.ProductFamily = "Load Balancer-Network"
	case TypeGateway:
		spec.ProductFamily = "Load Balancer-Gateway"
	default:
		spec.ProductFamily = ProductFamilyALB
	}

	return spec.Build(region, attrs)
}

func (h *ALBHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	d := map[string]string{}
	lbType := handler.GetStringAttr(attrs, "load_balancer_type")
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
		lbType := handler.GetStringAttr(attrs, "load_balancer_type")
		switch lbType {
		case TypeNetwork:
			rate = DefaultNLBHourlyCost
		case TypeGateway:
			rate = DefaultGWLBHourlyCost
		default:
			rate = DefaultALBHourlyCost
		}
	}
	return handler.HourlyCost(rate)
}
