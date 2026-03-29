package rds

import (
	"fmt"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// ClusterInstanceHandler handles aws_rds_cluster_instance cost estimation
type ClusterInstanceHandler struct{}

func (h *ClusterInstanceHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *ClusterInstanceHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceRDS
}

func (h *ClusterInstanceHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	instanceClass := handler.GetStringAttr(attrs, "instance_class")
	if instanceClass == "" {
		return nil, fmt.Errorf("instance_class not found")
	}

	engine := handler.GetStringAttr(attrs, "engine")
	if engine == "" {
		engine = DefaultAuroraEngine
	}

	databaseEngine := MapRDSEngine(engine)

	lb := &awskit.LookupBuilder{Service: pricing.ServiceRDS, ProductFamily: "Database Instance"}
	return lb.Build(region, map[string]string{
		"instanceType":   instanceClass,
		"databaseEngine": databaseEngine,
	}), nil
}

func (h *ClusterInstanceHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	d := map[string]string{}
	if v := handler.GetStringAttr(attrs, "instance_class"); v != "" {
		d["instance_class"] = v
	}
	if v := handler.GetStringAttr(attrs, "engine"); v != "" {
		d["engine"] = v
	}
	return d
}

func (h *ClusterInstanceHandler) CalculateCost(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
	return handler.HourlyCost(price.OnDemandUSD)
}
