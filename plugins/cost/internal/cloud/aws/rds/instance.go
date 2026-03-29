package rds

import (
	"errors"
	"fmt"
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
type InstanceHandler struct{}

func (h *InstanceHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *InstanceHandler) ServiceCode() pricing.ServiceID {
	return awskit.MustService(awskit.ServiceKeyRDS)
}

func (h *InstanceHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	instanceClass := handler.GetStringAttr(attrs, "instance_class")
	if instanceClass == "" {
		return nil, errors.New("instance_class not found")
	}

	engine := handler.GetStringAttr(attrs, "engine")
	if engine == "" {
		engine = DefaultEngine
	}

	// Map terraform engine to RDS database engine
	databaseEngine := MapRDSEngine(engine)

	// Deployment option
	deploymentOption := "Single-AZ"
	if handler.GetBoolAttr(attrs, "multi_az") {
		deploymentOption = "Multi-AZ"
	}

	lb := &awskit.LookupBuilder{Service: awskit.MustService(awskit.ServiceKeyRDS), ProductFamily: "Database Instance"}
	return lb.Build(region, map[string]string{
		"instanceType":     instanceClass,
		"databaseEngine":   databaseEngine,
		"deploymentOption": deploymentOption,
	}), nil
}

func (h *InstanceHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	d := map[string]string{}
	if v := handler.GetStringAttr(attrs, "instance_class"); v != "" {
		d["instance_class"] = v
	}
	if v := handler.GetStringAttr(attrs, "engine"); v != "" {
		d["engine"] = v
	}
	if handler.GetBoolAttr(attrs, "multi_az") {
		d["multi_az"] = "true"
	}
	if v := handler.GetFloatAttr(attrs, "allocated_storage"); v > 0 {
		d["storage_gb"] = fmt.Sprintf("%.0f", v)
	}
	return d
}

func (h *InstanceHandler) CalculateCost(price *pricing.Price, _ *pricing.PriceIndex, _ string, attrs map[string]any) (hourly, monthly float64) {
	hourly = price.OnDemandUSD
	monthly = hourly * handler.HoursPerMonth

	// Add storage cost
	storageType := handler.GetStringAttr(attrs, "storage_type")
	allocatedStorage := handler.GetFloatAttr(attrs, "allocated_storage")
	if allocatedStorage > 0 {
		storageCostPerGB := GetStorageCostPerGB(storageType)
		monthly += allocatedStorage * storageCostPerGB
	}

	// Add IOPS cost for provisioned IOPS storage types
	iops := handler.GetFloatAttr(attrs, "iops")
	if iops > 0 {
		switch storageType {
		case awskit.VolumeTypeIO1:
			monthly += iops * IOPSCostIO1PerMonth
		case awskit.VolumeTypeIO2:
			monthly += iops * IOPSCostIO2PerMonth
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
