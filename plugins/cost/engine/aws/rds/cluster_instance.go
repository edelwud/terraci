package rds

import (
	"fmt"

	aws "github.com/edelwud/terraci/plugins/cost/engine/aws"
	"github.com/edelwud/terraci/plugins/cost/engine/pricing"
)

// ClusterInstanceHandler handles aws_rds_cluster_instance cost estimation
type ClusterInstanceHandler struct{}

func (h *ClusterInstanceHandler) Category() aws.CostCategory { return aws.CostCategoryStandard }

func (h *ClusterInstanceHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceRDS
}

func (h *ClusterInstanceHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	instanceClass := aws.GetStringAttr(attrs, "instance_class")
	if instanceClass == "" {
		return nil, fmt.Errorf("instance_class not found")
	}

	engine := aws.GetStringAttr(attrs, "engine")
	if engine == "" {
		engine = DefaultAuroraEngine
	}

	databaseEngine := MapRDSEngine(engine)

	lb := &aws.LookupBuilder{Service: pricing.ServiceRDS, ProductFamily: "Database Instance"}
	return lb.Build(region, map[string]string{
		"instanceType":   instanceClass,
		"databaseEngine": databaseEngine,
	}), nil
}

func (h *ClusterInstanceHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	d := map[string]string{}
	if v := aws.GetStringAttr(attrs, "instance_class"); v != "" {
		d["instance_class"] = v
	}
	if v := aws.GetStringAttr(attrs, "engine"); v != "" {
		d["engine"] = v
	}
	return d
}

func (h *ClusterInstanceHandler) CalculateCost(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
	return aws.HourlyCost(price.OnDemandUSD)
}
