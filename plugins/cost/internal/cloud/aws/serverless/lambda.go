package serverless

import (
	"strconv"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// Lambda pricing constants
const (
	LambdaProvisionedConcurrencyCostPerGBSecond = 0.000004646
	LambdaDefaultMemoryMB                       = 128
	LambdaMemoryDivisor                         = 1024
	SecondsPerHour                              = 3600
)

// LambdaHandler handles aws_lambda_function cost estimation
// Note: Lambda pricing is usage-based (requests + duration)
// For fixed cost estimation, we estimate based on memory and assume average invocations
type LambdaHandler struct{}

func (h *LambdaHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *LambdaHandler) ServiceCode() pricing.ServiceID {
	return awskit.MustService(awskit.ServiceKeyLambda)
}

func (h *LambdaHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	desc := make(map[string]string)
	if v := handler.GetIntAttr(attrs, "memory_size"); v != 0 {
		desc["memory_mb"] = strconv.Itoa(v)
	}
	if v := handler.GetStringAttr(attrs, "runtime"); v != "" {
		desc["runtime"] = v
	}
	if v := handler.GetIntAttr(attrs, "provisioned_concurrent_executions"); v != 0 {
		desc["provisioned_concurrency"] = strconv.Itoa(v)
	}
	return desc
}

func (h *LambdaHandler) BuildLookup(region string, _ map[string]any) (*pricing.PriceLookup, error) {
	lb := &awskit.LookupBuilder{Service: awskit.MustService(awskit.ServiceKeyLambda), ProductFamily: "Serverless"}
	return lb.Build(region, map[string]string{
		"group": "AWS-Lambda-Duration",
	}), nil
}

func (h *LambdaHandler) CalculateCost(_ *pricing.Price, _ *pricing.PriceIndex, _ string, attrs map[string]any) (hourly, monthly float64) {
	// Lambda has complex pricing: requests + GB-seconds
	// For fixed cost, return 0 as it's usage-based
	// Could estimate based on provisioned concurrency if set
	provisionedConcurrency := handler.GetIntAttr(attrs, "provisioned_concurrent_executions")
	if provisionedConcurrency > 0 {
		memoryMB := handler.GetIntAttr(attrs, "memory_size")
		if memoryMB == 0 {
			memoryMB = LambdaDefaultMemoryMB
		}
		// Provisioned concurrency: $0.000004646 per GB-second
		gbSeconds := float64(provisionedConcurrency) * (float64(memoryMB) / LambdaMemoryDivisor) * SecondsPerHour
		rate := gbSeconds * LambdaProvisionedConcurrencyCostPerGBSecond
		return handler.HourlyCost(rate)
	}
	return 0, 0 // Usage-based, no fixed cost
}
