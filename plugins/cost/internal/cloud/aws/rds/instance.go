package rds

import (
	"errors"
	"strings"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
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
	IOPSCostGP3PerMonth = 0.00 // gp3 IOPS included (3000 baseline)

	AuroraStorageCostPerGB = 0.10

	// Default engine
	DefaultEngine       = "mysql"
	DefaultAuroraEngine = "aurora-mysql"
)

// InstanceHandler handles aws_db_instance cost estimation
type InstanceHandler struct {
	awskit.RuntimeDeps
}

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

func (h *InstanceHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *InstanceHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	parsed := parseInstanceAttrs(attrs)
	if parsed.InstanceClass == "" {
		return nil, errors.New("instance_class not found")
	}

	engine := parsed.Engine
	if engine == "" {
		engine = DefaultEngine
	}

	// Map terraform engine to RDS database engine
	databaseEngine := MapRDSEngine(engine)

	// Deployment option
	deploymentOption := "Single-AZ"
	if parsed.MultiAZ {
		deploymentOption = "Multi-AZ"
	}

	return h.RuntimeOrDefault().StandardLookupSpec(
		awskit.ServiceKeyRDS,
		"Database Instance",
		func(_ string, _ map[string]any) (map[string]string, error) {
			return map[string]string{
				"instanceType":     parsed.InstanceClass,
				"databaseEngine":   databaseEngine,
				"deploymentOption": deploymentOption,
			}, nil
		},
	).Build(region, attrs)
}

func (h *InstanceHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	parsed := parseInstanceAttrs(attrs)
	return awskit.NewDescribeBuilder().
		String("instance_class", parsed.InstanceClass).
		String("engine", parsed.Engine).
		Bool("multi_az", parsed.MultiAZ).
		Float("storage_gb", parsed.AllocatedStorage, "%.0f").
		Map()
}

func (h *InstanceHandler) CalculateCost(price *pricing.Price, _ *pricing.PriceIndex, _ string, attrs map[string]any) (hourly, monthly float64) {
	parsed := parseInstanceAttrs(attrs)
	hourly = price.OnDemandUSD
	monthly = hourly * handler.HoursPerMonth

	// Add storage cost
	if parsed.AllocatedStorage > 0 {
		storageCostPerGB := GetStorageCostPerGB(parsed.StorageType)
		monthly += parsed.AllocatedStorage * storageCostPerGB
	}

	// Add IOPS cost for provisioned IOPS storage types
	if parsed.IOPS > 0 {
		switch parsed.StorageType {
		case awskit.VolumeTypeIO1:
			monthly += parsed.IOPS * IOPSCostIO1PerMonth
		case awskit.VolumeTypeIO2:
			monthly += parsed.IOPS * IOPSCostIO2PerMonth
		}
		// gp3 IOPS are included in the base price for RDS (unlike EBS)
	}

	hourly = monthly / handler.HoursPerMonth
	return hourly, monthly
}

// MapRDSEngine maps terraform engine names to AWS pricing database engine names
func MapRDSEngine(engine string) string {
	engine = strings.ToLower(engine)
	switch {
	case strings.HasPrefix(engine, "aurora-mysql"):
		return "Aurora MySQL"
	case strings.HasPrefix(engine, "aurora-postgresql"):
		return "Aurora PostgreSQL"
	case strings.HasPrefix(engine, "aurora"):
		return "Aurora MySQL"
	case engine == "mysql":
		return "MySQL"
	case engine == "postgres", engine == "postgresql":
		return "PostgreSQL"
	case engine == "mariadb":
		return "MariaDB"
	case strings.HasPrefix(engine, "oracle"):
		return "Oracle"
	case strings.HasPrefix(engine, "sqlserver"):
		return "SQL Server"
	default:
		return "MySQL"
	}
}

// GetStorageCostPerGB returns estimated storage cost per GB-month
func GetStorageCostPerGB(storageType string) float64 {
	switch storageType {
	case awskit.VolumeTypeGP2, awskit.VolumeTypeGP3:
		return StorageCostGP2
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
