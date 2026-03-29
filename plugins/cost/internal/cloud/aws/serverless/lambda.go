package serverless

import (
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
type LambdaHandler struct {
	awskit.RuntimeDeps
}

func (h *LambdaHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *LambdaHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	return awskit.NewDescribeBuilder().
		Int("memory_mb", handler.GetIntAttr(attrs, "memory_size")).
		String("runtime", handler.GetStringAttr(attrs, "runtime")).
		Int("provisioned_concurrency", handler.GetIntAttr(attrs, "provisioned_concurrent_executions")).
		Map()
}

func (h *LambdaHandler) BuildLookup(region string, _ map[string]any) (*pricing.PriceLookup, error) {
	return h.RuntimeOrDefault().StandardLookupSpec(
		awskit.ServiceKeyLambda,
		"Serverless",
		func(_ string, _ map[string]any) (map[string]string, error) {
			return map[string]string{
				"group": "AWS-Lambda-Duration",
			}, nil
		},
	).Build(region, nil)
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
