package aws

import (
	"fmt"
	"strings"

	"github.com/edelwud/terraci/internal/cost/pricing"
)

// RDS pricing constants
const (
	// Storage costs per GB-month
	RDSStorageCostGP2      = 0.115
	RDSStorageCostGP3      = 0.115
	RDSStorageCostIO1      = 0.125
	RDSStorageCostStandard = 0.10
	RDSIOPSCostPerMonth    = 0.10
	AuroraStorageCostPerGB = 0.10

	// Default engine
	DefaultRDSEngine    = "mysql"
	DefaultAuroraEngine = "aurora-mysql"
)

// RDSInstanceHandler handles aws_db_instance cost estimation
type RDSInstanceHandler struct{}

func (h *RDSInstanceHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceRDS
}

func (h *RDSInstanceHandler) BuildLookup(region string, attrs map[string]interface{}) (*pricing.PriceLookup, error) {
	instanceClass := getStringAttr(attrs, "instance_class")
	if instanceClass == "" {
		return nil, fmt.Errorf("instance_class not found")
	}

	engine := getStringAttr(attrs, "engine")
	if engine == "" {
		engine = DefaultRDSEngine
	}

	// Map terraform engine to RDS database engine
	databaseEngine := mapRDSEngine(engine)

	// Deployment option
	deploymentOption := "Single-AZ"
	if getBoolAttr(attrs, "multi_az") {
		deploymentOption = "Multi-AZ"
	}

	regionName := pricing.RegionMapping[region]
	if regionName == "" {
		regionName = region
	}

	return &pricing.PriceLookup{
		ServiceCode:   pricing.ServiceRDS,
		Region:        region,
		ProductFamily: "Database Instance",
		Attributes: map[string]string{
			"instanceType":     instanceClass,
			"location":         regionName,
			"databaseEngine":   databaseEngine,
			"deploymentOption": deploymentOption,
		},
	}, nil
}

func (h *RDSInstanceHandler) CalculateCost(price *pricing.Price, attrs map[string]interface{}) (hourly, monthly float64) {
	hourly = price.OnDemandUSD
	monthly = hourly * HoursPerMonth

	// Add storage cost
	storageType := getStringAttr(attrs, "storage_type")
	allocatedStorage := getFloatAttr(attrs, "allocated_storage")
	if allocatedStorage > 0 {
		storageCostPerGB := getStorageCostPerGB(storageType)
		monthly += allocatedStorage * storageCostPerGB
	}

	// Add IOPS cost for io1
	if storageType == VolumeTypeIO1 {
		iops := getFloatAttr(attrs, "iops")
		if iops > 0 {
			monthly += iops * RDSIOPSCostPerMonth
		}
	}

	hourly = monthly / HoursPerMonth
	return hourly, monthly
}

// RDSClusterHandler handles aws_rds_cluster cost estimation (Aurora)
type RDSClusterHandler struct{}

func (h *RDSClusterHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceRDS
}

func (h *RDSClusterHandler) BuildLookup(region string, attrs map[string]interface{}) (*pricing.PriceLookup, error) {
	// Aurora cluster itself doesn't have hourly compute cost
	// Cost comes from cluster instances and storage
	// Return a lookup for storage pricing
	engine := getStringAttr(attrs, "engine")
	if engine == "" {
		engine = DefaultAuroraEngine
	}
	_ = engine // Engine used for validation only

	regionName := pricing.RegionMapping[region]
	if regionName == "" {
		regionName = region
	}

	return &pricing.PriceLookup{
		ServiceCode:   pricing.ServiceRDS,
		Region:        region,
		ProductFamily: "Database Storage",
		Attributes: map[string]string{
			"location":   regionName,
			"volumeType": "Aurora:StorageUsage",
			"usagetype":  region + "-Aurora:StorageUsage",
		},
	}, nil
}

func (h *RDSClusterHandler) CalculateCost(_ *pricing.Price, attrs map[string]interface{}) (hourly, monthly float64) {
	// Aurora storage is billed per GB-month
	// Estimate based on allocated storage or minimum
	allocatedStorage := getFloatAttr(attrs, "allocated_storage")
	if allocatedStorage == 0 {
		allocatedStorage = 10 // Minimum 10GB
	}

	// Aurora storage: ~$0.10 per GB-month
	monthly = allocatedStorage * AuroraStorageCostPerGB
	hourly = monthly / HoursPerMonth
	return hourly, monthly
}

// RDSClusterInstanceHandler handles aws_rds_cluster_instance cost estimation
type RDSClusterInstanceHandler struct{}

func (h *RDSClusterInstanceHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceRDS
}

func (h *RDSClusterInstanceHandler) BuildLookup(region string, attrs map[string]interface{}) (*pricing.PriceLookup, error) {
	instanceClass := getStringAttr(attrs, "instance_class")
	if instanceClass == "" {
		return nil, fmt.Errorf("instance_class not found")
	}

	engine := getStringAttr(attrs, "engine")
	if engine == "" {
		engine = DefaultAuroraEngine
	}

	databaseEngine := mapRDSEngine(engine)

	regionName := pricing.RegionMapping[region]
	if regionName == "" {
		regionName = region
	}

	return &pricing.PriceLookup{
		ServiceCode:   pricing.ServiceRDS,
		Region:        region,
		ProductFamily: "Database Instance",
		Attributes: map[string]string{
			"instanceType":   instanceClass,
			"location":       regionName,
			"databaseEngine": databaseEngine,
		},
	}, nil
}

func (h *RDSClusterInstanceHandler) CalculateCost(price *pricing.Price, _ map[string]interface{}) (hourly, monthly float64) {
	hourly = price.OnDemandUSD
	monthly = hourly * HoursPerMonth
	return hourly, monthly
}

// mapRDSEngine maps terraform engine names to AWS pricing database engine names
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

// getStorageCostPerGB returns estimated storage cost per GB-month
func getStorageCostPerGB(storageType string) float64 {
	switch storageType {
	case VolumeTypeGP2, VolumeTypeGP3:
		return RDSStorageCostGP2
	case VolumeTypeIO1:
		return RDSStorageCostIO1
	case VolumeTypeStandard:
		return RDSStorageCostStandard
	default:
		return RDSStorageCostGP2
	}
}
