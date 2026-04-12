package serverless

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

// Lambda pricing constants
const (
	LambdaProvisionedConcurrencyCostPerGBSecond = 0.000004646
	LambdaDefaultMemoryMB                       = 128
	LambdaMemoryDivisor                         = 1024
	SecondsPerHour                              = 3600
)

type lambdaAttrs struct {
	MemoryMB               int
	Runtime                string
	ProvisionedConcurrency int
}

func parseLambdaAttrs(attrs map[string]any) lambdaAttrs {
	return lambdaAttrs{
		MemoryMB:               costutil.GetIntAttr(attrs, "memory_size"),
		Runtime:                costutil.GetStringAttr(attrs, "runtime"),
		ProvisionedConcurrency: costutil.GetIntAttr(attrs, "provisioned_concurrent_executions"),
	}
}

// LambdaSpec declares aws_lambda_function cost estimation.
func LambdaSpec(deps awskit.RuntimeDeps) resourcespec.TypedSpec[lambdaAttrs] {
	return resourcespec.TypedSpec[lambdaAttrs]{
		Type:     resourcedef.ResourceType(awskit.ResourceLambdaFunction),
		Category: resourcedef.CostCategoryUsageBased,
		Parse:    parseLambdaAttrs,
		Lookup: &resourcespec.TypedLookupSpec[lambdaAttrs]{
			BuildFunc: func(region string, _ lambdaAttrs) (*pricing.PriceLookup, error) {
				return deps.RuntimeOrDefault().StandardLookupSpec(
					awskit.ServiceKeyLambda,
					"Serverless",
					func(_ string, _ map[string]any) (map[string]string, error) {
						return map[string]string{"group": "AWS-Lambda-Duration"}, nil
					},
				).Build(region, nil)
			},
		},
		Describe: &resourcespec.TypedDescribeSpec[lambdaAttrs]{
			BuildFunc: func(_ *pricing.Price, p lambdaAttrs) map[string]string {
				return awskit.NewDescribeBuilder().
					Int("memory_mb", p.MemoryMB).
					String("runtime", p.Runtime).
					Int("provisioned_concurrency", p.ProvisionedConcurrency).
					Map()
			},
		},
		Usage: &resourcespec.TypedUsagePricingSpec[lambdaAttrs]{
			EstimateFunc: func(_ string, p lambdaAttrs) model.UsageCostEstimate {
				if p.ProvisionedConcurrency > 0 {
					memoryMB := p.MemoryMB
					if memoryMB == 0 {
						memoryMB = LambdaDefaultMemoryMB
					}
					gbSeconds := float64(p.ProvisionedConcurrency) * (float64(memoryMB) / LambdaMemoryDivisor) * SecondsPerHour
					rate := gbSeconds * LambdaProvisionedConcurrencyCostPerGBSecond
					hourly, monthly := costutil.HourlyCost(rate)
					return model.UsageCostEstimate{
						HourlyCost:  hourly,
						MonthlyCost: monthly,
						Status:      model.ResourceEstimateStatusUsageEstimated,
						Detail:      "usage-based estimate derived from provisioned concurrency",
					}
				}
				return model.UsageCostEstimate{Status: model.ResourceEstimateStatusUsageUnknown}
			},
		},
	}
}
