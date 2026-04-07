package rds

import (
	"errors"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// ClusterInstanceHandler handles aws_rds_cluster_instance cost estimation
type ClusterInstanceHandler struct {
	awskit.RuntimeDeps
}

type clusterInstanceAttrs struct {
	InstanceClass string
	Engine        string
}

func parseClusterInstanceAttrs(attrs map[string]any) clusterInstanceAttrs {
	return clusterInstanceAttrs{
		InstanceClass: handler.GetStringAttr(attrs, "instance_class"),
		Engine:        handler.GetStringAttr(attrs, "engine"),
	}
}

func (h *ClusterInstanceHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *ClusterInstanceHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	parsed := parseClusterInstanceAttrs(attrs)
	if parsed.InstanceClass == "" {
		return nil, errors.New("instance_class not found")
	}

	engine := parsed.Engine
	if engine == "" {
		engine = DefaultAuroraEngine
	}

	databaseEngine := mapRDSEngine(engine)

	return h.RuntimeOrDefault().StandardLookupSpec(
		awskit.ServiceKeyRDS,
		"Database Instance",
		func(_ string, _ map[string]any) (map[string]string, error) {
			return map[string]string{
				"instanceType":   parsed.InstanceClass,
				"databaseEngine": databaseEngine,
			}, nil
		},
	).Build(region, attrs)
}

func (h *ClusterInstanceHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	parsed := parseClusterInstanceAttrs(attrs)
	return awskit.NewDescribeBuilder().
		String("instance_class", parsed.InstanceClass).
		String("engine", parsed.Engine).
		Map()
}

func (h *ClusterInstanceHandler) CalculateCost(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
	if price == nil {
		return 0, 0
	}
	return handler.HourlyCost(price.OnDemandUSD)
}
