package elb

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

const (
	DefaultClassicLBHourlyCost = 0.025
)

// ClassicHandler handles aws_elb (Classic) cost estimation
type ClassicHandler struct {
	awskit.RuntimeDeps
}

func (h *ClassicHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *ClassicHandler) BuildLookup(region string, _ map[string]any) (*pricing.PriceLookup, error) {
	return h.RuntimeOrDefault().StandardLookupSpec(
		awskit.ServiceKeyELB,
		"Load Balancer",
		func(_ string, _ map[string]any) (map[string]string, error) {
			return map[string]string{
				"usagetype": region + "-" + UsageType,
			}, nil
		},
	).Build(region, nil)
}

func (h *ClassicHandler) Describe(_ *pricing.Price, _ map[string]any) map[string]string {
	return map[string]string{"type": "classic"}
}

func (h *ClassicHandler) CalculateCost(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
	rate := price.OnDemandUSD
	if rate == 0 {
		rate = DefaultClassicLBHourlyCost
	}
	return handler.HourlyCost(rate)
}
