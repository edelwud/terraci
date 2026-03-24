package elb

import (
	aws "github.com/edelwud/terraci/plugins/cost/engine/aws"
	"github.com/edelwud/terraci/plugins/cost/engine/pricing"
)

const (
	DefaultClassicLBHourlyCost = 0.025
)

// ClassicHandler handles aws_elb (Classic) cost estimation
type ClassicHandler struct{}

func (h *ClassicHandler) Category() aws.CostCategory { return aws.CostCategoryStandard }

func (h *ClassicHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceELB
}

func (h *ClassicHandler) BuildLookup(region string, _ map[string]any) (*pricing.PriceLookup, error) {
	lb := &aws.LookupBuilder{Service: pricing.ServiceELB, ProductFamily: "Load Balancer"}
	return lb.Build(region, map[string]string{
		"usagetype": region + "-" + UsageType,
	}), nil
}

func (h *ClassicHandler) Describe(_ *pricing.Price, _ map[string]any) map[string]string {
	return map[string]string{"type": "classic"}
}

func (h *ClassicHandler) CalculateCost(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
	rate := price.OnDemandUSD
	if rate == 0 {
		rate = DefaultClassicLBHourlyCost
	}
	return aws.HourlyCost(rate)
}
