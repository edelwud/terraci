package rds

import (
	"errors"
	"strings"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

// RDS pricing constants
const (
	// Storage costs per GB-month
	StorageCostGP2      = 0.115
	StorageCostGP3      = 0.115
	StorageCostIO1      = 0.125
	StorageCostIO2      = 0.125
	StorageCostStandard = 0.10

	// IOPS costs per provisioned IOPS per month
	IOPSCostIO1PerMonth = 0.10
	IOPSCostIO2PerMonth = 0.10
	// gp3 IOPS are included in baseline (3000 IOPS) — no additional cost constant needed

	AuroraStorageCostPerGB = 0.10

	// Default engine
	DefaultEngine       = "mysql"
	DefaultAuroraEngine = "aurora-mysql"

	// Deployment options
	DeploymentSingleAZ = "Single-AZ"
	DeploymentMultiAZ  = "Multi-AZ"

	// Pricing engine names
	EngineMySQL = "MySQL"
)

type instanceAttrs struct {
	InstanceClass    string
	Engine           string
	StorageType      string
	AllocatedStorage float64
	IOPS             float64
	MultiAZ          bool
}

func parseInstanceAttrs(attrs map[string]any) instanceAttrs {
	return instanceAttrs{
		InstanceClass:    handler.GetStringAttr(attrs, "instance_class"),
		Engine:           handler.GetStringAttr(attrs, "engine"),
		StorageType:      handler.GetStringAttr(attrs, "storage_type"),
		AllocatedStorage: handler.GetFloatAttr(attrs, "allocated_storage"),
		IOPS:             handler.GetFloatAttr(attrs, "iops"),
		MultiAZ:          handler.GetBoolAttr(attrs, "multi_az"),
	}
}

// InstanceSpec declares aws_db_instance cost estimation.
func InstanceSpec(deps awskit.RuntimeDeps) resourcespec.ResourceSpec {
	return resourcespec.ResourceSpec{
		Type:     handler.ResourceType(awskit.ResourceDBInstance),
		Category: handler.CostCategoryStandard,
		Lookup: &resourcespec.LookupSpec{
			BuildFunc: func(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
				parsed := parseInstanceAttrs(attrs)
				if parsed.InstanceClass == "" {
					return nil, errors.New("instance_class not found")
				}

				engine := parsed.Engine
				if engine == "" {
					engine = DefaultEngine
				}
				deploymentOption := DeploymentSingleAZ
				if parsed.MultiAZ {
					deploymentOption = DeploymentMultiAZ
				}

				return deps.RuntimeOrDefault().StandardLookupSpec(
					awskit.ServiceKeyRDS,
					"Database Instance",
					func(_ string, _ map[string]any) (map[string]string, error) {
						return map[string]string{
							"instanceType":     parsed.InstanceClass,
							"databaseEngine":   mapRDSEngine(engine),
							"deploymentOption": deploymentOption,
						}, nil
					},
				).Build(region, attrs)
			},
		},
		Describe: &resourcespec.DescribeSpec{
			BuildFunc: func(_ *pricing.Price, attrs map[string]any) map[string]string {
				parsed := parseInstanceAttrs(attrs)
				return awskit.NewDescribeBuilder().
					String("instance_class", parsed.InstanceClass).
					String("engine", parsed.Engine).
					Bool("multi_az", parsed.MultiAZ).
					Float("storage_gb", parsed.AllocatedStorage, "%.0f").
					Map()
			},
		},
		Standard: &resourcespec.StandardPricingSpec{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, attrs map[string]any) (hourly, monthly float64) {
				if price == nil {
					return 0, 0
				}
				parsed := parseInstanceAttrs(attrs)
				hourly = price.OnDemandUSD
				monthly = hourly * handler.HoursPerMonth
				if parsed.AllocatedStorage > 0 {
					monthly += parsed.AllocatedStorage * getStorageCostPerGB(parsed.StorageType)
				}
				if parsed.IOPS > 0 {
					switch parsed.StorageType {
					case awskit.VolumeTypeIO1:
						monthly += parsed.IOPS * IOPSCostIO1PerMonth
					case awskit.VolumeTypeIO2:
						monthly += parsed.IOPS * IOPSCostIO2PerMonth
					}
				}
				return monthly / handler.HoursPerMonth, monthly
			},
		},
	}
}

// mapRDSEngine maps terraform engine names to AWS pricing database engine names.
func mapRDSEngine(engine string) string {
	engine = strings.ToLower(engine)
	switch {
	case strings.HasPrefix(engine, "aurora-mysql"):
		return "Aurora MySQL"
	case strings.HasPrefix(engine, "aurora-postgresql"):
		return "Aurora PostgreSQL"
	case strings.HasPrefix(engine, "aurora"):
		return "Aurora MySQL"
	case engine == "mysql":
		return EngineMySQL
	case engine == "postgres", engine == "postgresql":
		return "PostgreSQL"
	case engine == "mariadb":
		return "MariaDB"
	case strings.HasPrefix(engine, "oracle"):
		return "Oracle"
	case strings.HasPrefix(engine, "sqlserver"):
		return "SQL Server"
	default:
		return EngineMySQL
	}
}

// getStorageCostPerGB returns estimated storage cost per GB-month.
func getStorageCostPerGB(storageType string) float64 {
	switch storageType {
	case awskit.VolumeTypeGP2:
		return StorageCostGP2
	case awskit.VolumeTypeGP3:
		return StorageCostGP3
	case awskit.VolumeTypeIO1:
		return StorageCostIO1
	case awskit.VolumeTypeIO2:
		return StorageCostIO2
	case awskit.VolumeTypeStandard:
		return StorageCostStandard
	default:
		return StorageCostGP2
	}
}
