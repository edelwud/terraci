package rds

import (
	"strings"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

type clusterAttrs struct {
	Engine           string
	AllocatedStorage float64
}

func parseClusterAttrs(attrs map[string]any) clusterAttrs {
	return clusterAttrs{
		Engine:           costutil.GetStringAttr(attrs, "engine"),
		AllocatedStorage: costutil.GetFloatAttr(attrs, "allocated_storage"),
	}
}

// isAuroraEngine returns true when the engine string is an Aurora variant.
func isAuroraEngine(engine string) bool {
	return strings.HasPrefix(strings.ToLower(engine), "aurora")
}

// ClusterSpec declares aws_rds_cluster cost estimation.
func ClusterSpec(deps awskit.RuntimeDeps) resourcespec.TypedSpec[clusterAttrs] {
	return resourcespec.TypedSpec[clusterAttrs]{
		Type:     resourcedef.ResourceType(awskit.ResourceRDSCluster),
		Category: resourcedef.CostCategoryStandard,
		Parse:    parseClusterAttrs,
		Lookup: &resourcespec.TypedLookupSpec[clusterAttrs]{
			BuildFunc: func(region string, p clusterAttrs) (*pricing.PriceLookup, error) {
				engine := p.Engine
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
				).Build(region, nil)
			},
		},
		Describe: &resourcespec.TypedDescribeSpec[clusterAttrs]{
			BuildFunc: func(_ *pricing.Price, p clusterAttrs) map[string]string {
				return awskit.NewDescribeBuilder().
					String("engine", p.Engine).
					Float("storage_gb", p.AllocatedStorage, "%.0f").
					Map()
			},
		},
		Standard: &resourcespec.TypedStandardPricingSpec[clusterAttrs]{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, p clusterAttrs) (hourly, monthly float64) {
				allocatedStorage := p.AllocatedStorage
				if allocatedStorage == 0 {
					allocatedStorage = 10
				}
				costPerGB := AuroraStorageCostPerGB
				if price != nil && price.OnDemandUSD > 0 {
					costPerGB = price.OnDemandUSD
				}
				monthly = allocatedStorage * costPerGB
				return monthly / costutil.HoursPerMonth, monthly
			},
		},
	}
}
