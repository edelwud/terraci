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

func (h *ClusterInstanceHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *ClusterInstanceHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	instanceClass := handler.GetStringAttr(attrs, "instance_class")
	if instanceClass == "" {
		return nil, errors.New("instance_class not found")
	}

	engine := handler.GetStringAttr(attrs, "engine")
	if engine == "" {
		engine = DefaultAuroraEngine
	}

	databaseEngine := MapRDSEngine(engine)

	return h.RuntimeOrDefault().StandardLookupSpec(
		awskit.ServiceKeyRDS,
		"Database Instance",
		func(_ string, _ map[string]any) (map[string]string, error) {
			return map[string]string{
				"instanceType":   instanceClass,
				"databaseEngine": databaseEngine,
			}, nil
		},
	).Build(region, attrs)
}

func (h *ClusterInstanceHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	return awskit.NewDescribeBuilder().
		String("instance_class", handler.GetStringAttr(attrs, "instance_class")).
		String("engine", handler.GetStringAttr(attrs, "engine")).
		Map()
}

func (h *ClusterInstanceHandler) CalculateCost(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
	return handler.HourlyCost(price.OnDemandUSD)
}
