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
func ClusterInstanceSpec(deps awskit.RuntimeDeps) resourcespec.ResourceSpec {
	return resourcespec.ResourceSpec{
		Type:     resourcedef.ResourceType(awskit.ResourceRDSClusterInstance),
		Category: resourcedef.CostCategoryStandard,
		Lookup: &resourcespec.LookupSpec{
			BuildFunc: func(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
				parsed := parseClusterInstanceAttrs(attrs)
				if parsed.InstanceClass == "" {
					return nil, errors.New("instance_class not found")
				}

				engine := parsed.Engine
				if engine == "" {
					engine = DefaultAuroraEngine
				}
				databaseEngine := mapRDSEngine(engine)

				return deps.RuntimeOrDefault().StandardLookupSpec(
					awskit.ServiceKeyRDS,
					"Database Instance",
					func(_ string, _ map[string]any) (map[string]string, error) {
						return map[string]string{
							"instanceType":   parsed.InstanceClass,
							"databaseEngine": databaseEngine,
						}, nil
					},
				).Build(region, attrs)
			},
		},
		Describe: &resourcespec.DescribeSpec{
			Fields: []resourcespec.DescribeField{
				{Key: "instance_class", Value: resourcespec.StringAttr("instance_class"), OmitEmpty: true},
				{Key: "engine", Value: resourcespec.StringAttr("engine"), OmitEmpty: true},
			},
		},
		Standard: &resourcespec.StandardPricingSpec{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
				if price == nil {
					return 0, 0
				}
				return costutil.HourlyCost(price.OnDemandUSD)
			},
		},
	}
}
