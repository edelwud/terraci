package rds

import (
	"strings"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

type clusterAttrs struct {
	Engine           string
	AllocatedStorage float64
}

func parseClusterAttrs(attrs map[string]any) clusterAttrs {
	return clusterAttrs{
		Engine:           handler.GetStringAttr(attrs, "engine"),
		AllocatedStorage: handler.GetFloatAttr(attrs, "allocated_storage"),
	}
}

// isAuroraEngine returns true when the engine string is an Aurora variant.
func isAuroraEngine(engine string) bool {
	return strings.HasPrefix(strings.ToLower(engine), "aurora")
}

// ClusterSpec declares aws_rds_cluster cost estimation.
func ClusterSpec(deps awskit.RuntimeDeps) resourcespec.ResourceSpec {
	return resourcespec.ResourceSpec{
		Type:     handler.ResourceType(awskit.ResourceRDSCluster),
		Category: handler.CostCategoryStandard,
		Lookup: &resourcespec.LookupSpec{
			BuildFunc: func(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
				parsed := parseClusterAttrs(attrs)
				engine := parsed.Engine
				if engine == "" {
					engine = DefaultAuroraEngine
				}
				if !isAuroraEngine(engine) {
					return nil, nil
				}

				runtime := deps.RuntimeOrDefault()
				return runtime.StandardLookupSpec(
					awskit.ServiceKeyRDS,
					"Database Storage",
					func(region string, _ map[string]any) (map[string]string, error) {
						prefix := runtime.ResolveUsagePrefix(region)
						return map[string]string{
							"volumeType": "Aurora:StorageUsage",
							"usagetype":  prefix + "-Aurora:StorageUsage",
						}, nil
					},
				).Build(region, attrs)
			},
		},
		Describe: &resourcespec.DescribeSpec{
			BuildFunc: func(_ *pricing.Price, attrs map[string]any) map[string]string {
				parsed := parseClusterAttrs(attrs)
				return awskit.NewDescribeBuilder().
					String("engine", parsed.Engine).
					Float("storage_gb", parsed.AllocatedStorage, "%.0f").
					Map()
			},
		},
		Standard: &resourcespec.StandardPricingSpec{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, attrs map[string]any) (hourly, monthly float64) {
				parsed := parseClusterAttrs(attrs)
				allocatedStorage := parsed.AllocatedStorage
				if allocatedStorage == 0 {
					allocatedStorage = 10
				}
				costPerGB := AuroraStorageCostPerGB
				if price != nil && price.OnDemandUSD > 0 {
					costPerGB = price.OnDemandUSD
				}
				monthly = allocatedStorage * costPerGB
				return monthly / handler.HoursPerMonth, monthly
			},
		},
	}
}
