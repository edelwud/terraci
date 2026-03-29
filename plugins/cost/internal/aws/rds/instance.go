package rds

import (
	"fmt"
	"strings"

	aws "github.com/edelwud/terraci/plugins/cost/internal/aws"
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
type InstanceHandler struct{}

func (h *InstanceHandler) Category() aws.CostCategory { return aws.CostCategoryStandard }

func (h *InstanceHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceRDS
}

func (h *InstanceHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	instanceClass := aws.GetStringAttr(attrs, "instance_class")
	if instanceClass == "" {
		return nil, fmt.Errorf("instance_class not found")
	}

	engine := aws.GetStringAttr(attrs, "engine")
	if engine == "" {
		engine = DefaultEngine
	}

	// Map terraform engine to RDS database engine
	databaseEngine := MapRDSEngine(engine)

	// Deployment option
	deploymentOption := "Single-AZ"
	if aws.GetBoolAttr(attrs, "multi_az") {
		deploymentOption = "Multi-AZ"
	}

	lb := &aws.LookupBuilder{Service: pricing.ServiceRDS, ProductFamily: "Database Instance"}
	return lb.Build(region, map[string]string{
		"instanceType":     instanceClass,
		"databaseEngine":   databaseEngine,
		"deploymentOption": deploymentOption,
	}), nil
}

func (h *InstanceHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	d := map[string]string{}
	if v := aws.GetStringAttr(attrs, "instance_class"); v != "" {
		d["instance_class"] = v
	}
	if v := aws.GetStringAttr(attrs, "engine"); v != "" {
		d["engine"] = v
	}
	if aws.GetBoolAttr(attrs, "multi_az") {
		d["multi_az"] = "true"
	}
	if v := aws.GetFloatAttr(attrs, "allocated_storage"); v > 0 {
		d["storage_gb"] = fmt.Sprintf("%.0f", v)
	}
	return d
}

func (h *InstanceHandler) CalculateCost(price *pricing.Price, _ *pricing.PriceIndex, _ string, attrs map[string]any) (hourly, monthly float64) {
	hourly = price.OnDemandUSD
	monthly = hourly * aws.HoursPerMonth

	// Add storage cost
	storageType := aws.GetStringAttr(attrs, "storage_type")
	allocatedStorage := aws.GetFloatAttr(attrs, "allocated_storage")
	if allocatedStorage > 0 {
		storageCostPerGB := GetStorageCostPerGB(storageType)
		monthly += allocatedStorage * storageCostPerGB
	}

	// Add IOPS cost for provisioned IOPS storage types
	iops := aws.GetFloatAttr(attrs, "iops")
	if iops > 0 {
		switch storageType {
		case aws.VolumeTypeIO1:
			monthly += iops * IOPSCostIO1PerMonth
		case aws.VolumeTypeIO2:
			monthly += iops * IOPSCostIO2PerMonth
		}
		// gp3 IOPS are included in the base price for RDS (unlike EBS)
	}

	hourly = monthly / aws.HoursPerMonth
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
	case aws.VolumeTypeGP2, aws.VolumeTypeGP3:
		return StorageCostGP2
	case aws.VolumeTypeIO1:
		return StorageCostIO1
	case aws.VolumeTypeIO2:
		return StorageCostIO2
	case aws.VolumeTypeStandard:
		return StorageCostStandard
	default:
		return StorageCostGP2
	}
}
