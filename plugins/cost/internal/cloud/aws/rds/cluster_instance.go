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

				return deps.RuntimeOrDefault().
					NewLookupBuilder(awskit.ServiceKeyRDS, "Database Instance").
					Attr("instanceType", p.InstanceClass).
					Attr("databaseEngine", mapRDSEngine(engine)).
					Build(region), nil
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
				return awskit.NewCostBuilder().Hourly().Calc(price, nil, "")
			},
		},
	}
}
