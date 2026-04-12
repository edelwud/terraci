package rds

import (
	"errors"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

type clusterInstanceAttrs struct {
	InstanceClass string
	Engine        string
}

func parseClusterInstanceAttrs(attrs map[string]any) clusterInstanceAttrs {
	return clusterInstanceAttrs{
		InstanceClass: costutil.GetStringAttr(attrs, "instance_class"),
		Engine:        costutil.GetStringAttr(attrs, "engine"),
	}
}

// ClusterInstanceSpec declares aws_rds_cluster_instance cost estimation.
func ClusterInstanceSpec(deps awskit.RuntimeDeps) resourcespec.TypedSpec[clusterInstanceAttrs] {
	return resourcespec.TypedSpec[clusterInstanceAttrs]{
		Type:     resourcedef.ResourceType(awskit.ResourceRDSClusterInstance),
		Category: resourcedef.CostCategoryStandard,
		Parse:    parseClusterInstanceAttrs,
		Lookup: &resourcespec.TypedLookupSpec[clusterInstanceAttrs]{
			BuildFunc: func(region string, p clusterInstanceAttrs) (*pricing.PriceLookup, error) {
				if p.InstanceClass == "" {
					return nil, errors.New("instance_class not found")
				}

				engine := p.Engine
				if engine == "" {
					engine = DefaultAuroraEngine
				}
				databaseEngine := mapRDSEngine(engine)

				return deps.RuntimeOrDefault().StandardLookupSpec(
					awskit.ServiceKeyRDS,
					"Database Instance",
					func(_ string, _ map[string]any) (map[string]string, error) {
						return map[string]string{
							"instanceType":   p.InstanceClass,
							"databaseEngine": databaseEngine,
						}, nil
					},
				).Build(region, nil)
			},
		},
		Describe: &resourcespec.TypedDescribeSpec[clusterInstanceAttrs]{
			BuildFunc: func(_ *pricing.Price, p clusterInstanceAttrs) map[string]string {
				return awskit.NewDescribeBuilder().
					String("instance_class", p.InstanceClass).
					String("engine", p.Engine).
					Map()
			},
		},
		Standard: &resourcespec.TypedStandardPricingSpec[clusterInstanceAttrs]{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ clusterInstanceAttrs) (hourly, monthly float64) {
				if price == nil {
					return 0, 0
				}
				return costutil.HourlyCost(price.OnDemandUSD)
			},
		},
	}
}
