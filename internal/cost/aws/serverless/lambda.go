package serverless

import (
	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
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

func (h *LambdaHandler) Category() aws.CostCategory { return aws.CostCategoryStandard }

func (h *LambdaHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceLambda
}

func (h *LambdaHandler) BuildLookup(region string, _ map[string]any) (*pricing.PriceLookup, error) {
	lb := &aws.LookupBuilder{Service: pricing.ServiceLambda, ProductFamily: "Serverless"}
	return lb.Build(region, map[string]string{
		"group": "AWS-Lambda-Duration",
	}), nil
}

func (h *LambdaHandler) CalculateCost(_ *pricing.Price, attrs map[string]any) (hourly, monthly float64) {
	// Lambda has complex pricing: requests + GB-seconds
	// For fixed cost, return 0 as it's usage-based
	// Could estimate based on provisioned concurrency if set
	provisionedConcurrency := aws.GetIntAttr(attrs, "provisioned_concurrent_executions")
	if provisionedConcurrency > 0 {
		memoryMB := aws.GetIntAttr(attrs, "memory_size")
		if memoryMB == 0 {
			memoryMB = LambdaDefaultMemoryMB
		}
		// Provisioned concurrency: $0.000004646 per GB-second
		gbSeconds := float64(provisionedConcurrency) * (float64(memoryMB) / LambdaMemoryDivisor) * SecondsPerHour
		rate := gbSeconds * LambdaProvisionedConcurrencyCostPerGBSecond
		return aws.HourlyCost(rate)
	}
	return 0, 0 // Usage-based, no fixed cost
}
