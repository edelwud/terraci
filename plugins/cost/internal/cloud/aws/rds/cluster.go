package rds

import (
	"strings"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

const deploymentMultiAZReadableStandbys = "Multi-AZ (readable standbys)"

// RDS cluster storage types repeated across this package and tests.
const (
	storageTypeAurora      = "aurora"
	storageTypeAuroraIOOpt = "aurora-iopt1"
)

type clusterAttrs struct {
	Engine           string
	StorageType      string
	AllocatedStorage float64
	MultiAZ          bool
}

func parseClusterAttrs(attrs map[string]any) clusterAttrs {
	engine := costutil.GetStringAttr(attrs, "engine")
	storageType := costutil.GetStringAttr(attrs, "storage_type")
	// db_cluster_instance_class is set only for Multi-AZ DB clusters (non-Aurora).
	multiAZ := costutil.GetStringAttr(attrs, "db_cluster_instance_class") != ""

	if storageType == "" {
		if strings.HasPrefix(strings.ToLower(engine), storageTypeAurora) || engine == "" {
			storageType = storageTypeAurora
		} else {
			storageType = awskit.VolumeTypeGP3
		}
	}
	return clusterAttrs{
		Engine:           engine,
		StorageType:      storageType,
		AllocatedStorage: costutil.GetFloatAttr(attrs, "allocated_storage"),
		MultiAZ:          multiAZ,
	}
}

func clusterStorageFallback(storageType string) float64 {
	switch storageType {
	case storageTypeAurora:
		return AuroraStorageCostPerGB
	case storageTypeAuroraIOOpt:
		return AuroraIOOptStorageCostPerGB
	default:
		return getStorageCostPerGB(storageType)
	}
}

// ClusterSpec declares aws_rds_cluster cost estimation.
func ClusterSpec(deps awskit.RuntimeDeps) resourcespec.TypedSpec[clusterAttrs] {
	return resourcespec.TypedSpec[clusterAttrs]{
		Type:     resourcedef.ResourceType(awskit.ResourceRDSCluster),
		Category: resourcedef.CostCategoryStandard,
		Parse:    parseClusterAttrs,
		Lookup: &resourcespec.TypedLookupSpec[clusterAttrs]{
			BuildFunc: func(region string, p clusterAttrs) (*pricing.PriceLookup, error) {
				return deps.RuntimeOrDefault().
					NewLookupBuilder(awskit.ServiceKeyRDS, "Database Storage").
					AttrIf(p.MultiAZ, "databaseEngine", mapRDSEngine(p.Engine)).
					AttrIf(p.MultiAZ, "deploymentOption", deploymentMultiAZReadableStandbys).
					AttrMatch("volumeType", p.StorageType, "", map[string]string{
						storageTypeAurora:      "Aurora:StorageUsage",
						storageTypeAuroraIOOpt: "Aurora:StorageIOUsage",
						awskit.VolumeTypeGP3:   "General Purpose-GP3",
						awskit.VolumeTypeIO1:   "Provisioned IOPS",
						awskit.VolumeTypeIO2:   "Provisioned IOPS",
					}).
					UsageType(region, awskit.MatchString(p.StorageType, "", map[string]string{
						storageTypeAurora:      "Aurora:StorageUsage",
						storageTypeAuroraIOOpt: "Aurora:StorageIOUsage",
						awskit.VolumeTypeGP3:   "RDS:Multi-AZCluster-GP3-Storage",
						awskit.VolumeTypeIO1:   "RDS:Multi-AZCluster-PIOPS-Storage",
						awskit.VolumeTypeIO2:   "RDS:Multi-AZCluster-PIOPS-Storage",
					})).
					Build(region), nil
			},
		},
		Describe: &resourcespec.TypedDescribeSpec[clusterAttrs]{
			BuildFunc: func(_ *pricing.Price, p clusterAttrs) map[string]string {
				return awskit.NewDescribeBuilder().
					String("engine", p.Engine).
					String("storage_type", p.StorageType).
					BoolIf(p.MultiAZ, "multi_az", p.MultiAZ).
					Float("storage_gb", p.AllocatedStorage, "%.0f").
					Map()
			},
		},
		Standard: &resourcespec.TypedStandardPricingSpec[clusterAttrs]{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, p clusterAttrs) (hourly, monthly float64) {
				storage := p.AllocatedStorage
				if storage == 0 {
					storage = 10
				}
				return awskit.NewCostBuilder().
					PerUnit(storage).
					Fallback(clusterStorageFallback(p.StorageType)).
					Calc(price, nil, "")
			},
		},
	}
}
