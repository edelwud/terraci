package rds

import (
	"errors"
	"strings"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
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

	AuroraStorageCostPerGB      = 0.10
	AuroraIOOptStorageCostPerGB = 0.225

	// Default engine
	DefaultEngine       = "mysql"
	DefaultAuroraEngine = "aurora-mysql"

	// Deployment options
	DeploymentSingleAZ = "Single-AZ"
	DeploymentMultiAZ  = "Multi-AZ"

	// Pricing engine names
	EngineMySQL            = "MySQL"
	EngineAuroraMySQL      = "Aurora MySQL"
	EngineAuroraPostgreSQL = "Aurora PostgreSQL"
	EnginePostgreSQL       = "PostgreSQL"
	EngineMariaDB          = "MariaDB"
	EngineOracle           = "Oracle"
	EngineSQLServer        = "SQL Server"
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
		InstanceClass:    costutil.GetStringAttr(attrs, "instance_class"),
		Engine:           costutil.GetStringAttr(attrs, "engine"),
		StorageType:      costutil.GetStringAttr(attrs, "storage_type"),
		AllocatedStorage: costutil.GetFloatAttr(attrs, "allocated_storage"),
		IOPS:             costutil.GetFloatAttr(attrs, "iops"),
		MultiAZ:          costutil.GetBoolAttr(attrs, "multi_az"),
	}
}

// InstanceSpec declares aws_db_instance cost estimation.
func InstanceSpec(deps awskit.RuntimeDeps) resourcespec.TypedSpec[instanceAttrs] {
	return resourcespec.TypedSpec[instanceAttrs]{
		Type:     resourcedef.ResourceType(awskit.ResourceDBInstance),
		Category: resourcedef.CostCategoryStandard,
		Parse:    parseInstanceAttrs,
		Lookup: &resourcespec.TypedLookupSpec[instanceAttrs]{
			BuildFunc: func(region string, p instanceAttrs) (*pricing.PriceLookup, error) {
				if p.InstanceClass == "" {
					return nil, errors.New("instance_class not found")
				}

				engine := p.Engine
				if engine == "" {
					engine = DefaultEngine
				}
				deploymentOption := DeploymentSingleAZ
				if p.MultiAZ {
					deploymentOption = DeploymentMultiAZ
				}

				return deps.RuntimeOrDefault().
					NewLookupBuilder(awskit.ServiceKeyRDS, "Database Instance").
					Attr("instanceType", p.InstanceClass).
					Attr("databaseEngine", mapRDSEngine(engine)).
					Attr("deploymentOption", deploymentOption).
					Build(region), nil
			},
		},
		Describe: &resourcespec.TypedDescribeSpec[instanceAttrs]{
			BuildFunc: func(_ *pricing.Price, p instanceAttrs) map[string]string {
				return awskit.NewDescribeBuilder().
					String("instance_class", p.InstanceClass).
					String("engine", p.Engine).
					Bool("multi_az", p.MultiAZ).
					Float("storage_gb", p.AllocatedStorage, "%.0f").
					Map()
			},
		},
		Standard: &resourcespec.TypedStandardPricingSpec[instanceAttrs]{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, p instanceAttrs) (hourly, monthly float64) {
				return awskit.NewCostBuilder().
					Hourly().
					Match(p.StorageType, []awskit.Charge{awskit.NewCharge(p.AllocatedStorage).Fixed(StorageCostGP2)}, map[string][]awskit.Charge{
						awskit.VolumeTypeGP2:      {awskit.NewCharge(p.AllocatedStorage).Fixed(StorageCostGP2)},
						awskit.VolumeTypeGP3:      {awskit.NewCharge(p.AllocatedStorage).Fixed(StorageCostGP3)},
						awskit.VolumeTypeIO1:      {awskit.NewCharge(p.AllocatedStorage).Fixed(StorageCostIO1), awskit.NewCharge(p.IOPS).Fixed(IOPSCostIO1PerMonth)},
						awskit.VolumeTypeIO2:      {awskit.NewCharge(p.AllocatedStorage).Fixed(StorageCostIO2), awskit.NewCharge(p.IOPS).Fixed(IOPSCostIO2PerMonth)},
						awskit.VolumeTypeStandard: {awskit.NewCharge(p.AllocatedStorage).Fixed(StorageCostStandard)},
					}).
					Calc(price, nil, "")
			},
		},
	}
}

// mapRDSEngine maps terraform engine names to AWS pricing database engine names.
func mapRDSEngine(engine string) string {
	engine = strings.ToLower(engine)
	switch {
	case strings.HasPrefix(engine, "aurora-mysql"):
		return EngineAuroraMySQL
	case strings.HasPrefix(engine, "aurora-postgresql"):
		return EngineAuroraPostgreSQL
	case strings.HasPrefix(engine, "aurora"):
		return EngineAuroraMySQL
	case engine == "mysql":
		return EngineMySQL
	case engine == "postgres", engine == "postgresql":
		return EnginePostgreSQL
	case engine == "mariadb":
		return EngineMariaDB
	case strings.HasPrefix(engine, "oracle"):
		return EngineOracle
	case strings.HasPrefix(engine, "sqlserver"):
		return EngineSQLServer
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
