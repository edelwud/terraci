package aws

import (
	"fmt"

	"github.com/edelwud/terraci/internal/cost/pricing"
)

// AWS pricing constants
const (
	// HoursPerMonth is the average number of hours in a month for cost calculations
	HoursPerMonth = 730

	// Default volume types
	VolumeTypeGP2         = "gp2"
	VolumeTypeGP3         = "gp3"
	VolumeTypeIO1         = "io1"
	VolumeTypeIO2         = "io2"
	VolumeTypeST1         = "st1"
	VolumeTypeSC1         = "sc1"
	VolumeTypeStandard    = "standard"
	VolumeTypeMagnetic    = "Magnetic"
	VolumeTypeGeneral     = "General Purpose"
	VolumeTypeProvisioned = "Provisioned IOPS"

	// Default pricing fallbacks (us-east-1)
	DefaultIO1IOPSCostPerMonth    = 0.065
	DefaultGP3ThroughputCostPerMB = 0.040
	DefaultGP3FreeThroughputMBps  = 125
	DefaultNATGatewayHourlyCost   = 0.045
)

// EC2InstanceHandler handles aws_instance cost estimation
type EC2InstanceHandler struct{}

func (h *EC2InstanceHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceEC2
}

func (h *EC2InstanceHandler) BuildLookup(region string, attrs map[string]interface{}) (*pricing.PriceLookup, error) {
	instanceType := getStringAttr(attrs, "instance_type")
	if instanceType == "" {
		return nil, fmt.Errorf("instance_type not found")
	}

	tenancy := getStringAttr(attrs, "tenancy")
	switch tenancy {
	case "", "default":
		tenancy = "Shared"
	case "dedicated":
		tenancy = "Dedicated"
	case "host":
		tenancy = "Host"
	}

	// Determine OS
	operatingSystem := "Linux"
	if ami := getStringAttr(attrs, "ami"); ami != "" {
		// Could check AMI patterns, but default to Linux for now
		operatingSystem = "Linux"
	}

	regionName := pricing.RegionMapping[region]
	if regionName == "" {
		regionName = region
	}

	return &pricing.PriceLookup{
		ServiceCode:   pricing.ServiceEC2,
		Region:        region,
		ProductFamily: "Compute Instance",
		Attributes: map[string]string{
			"instanceType":    instanceType,
			"location":        regionName,
			"tenancy":         tenancy,
			"operatingSystem": operatingSystem,
			"preInstalledSw":  "NA",
			"capacitystatus":  "Used",
		},
	}, nil
}

func (h *EC2InstanceHandler) CalculateCost(price *pricing.Price, _ map[string]interface{}) (hourly, monthly float64) {
	hourly = price.OnDemandUSD
	monthly = hourly * HoursPerMonth
	return hourly, monthly
}

// EBSVolumeHandler handles aws_ebs_volume cost estimation
type EBSVolumeHandler struct{}

func (h *EBSVolumeHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceEC2
}

func (h *EBSVolumeHandler) BuildLookup(region string, attrs map[string]interface{}) (*pricing.PriceLookup, error) {
	volumeType := getStringAttr(attrs, "type")
	if volumeType == "" {
		volumeType = VolumeTypeGP2 // Default
	}

	// Map terraform volume types to AWS pricing attributes
	volumeAPIName := map[string]string{
		VolumeTypeGP2:      VolumeTypeGeneral,
		VolumeTypeGP3:      VolumeTypeGeneral,
		VolumeTypeIO1:      VolumeTypeProvisioned,
		VolumeTypeIO2:      VolumeTypeProvisioned,
		VolumeTypeST1:      "Throughput Optimized HDD",
		VolumeTypeSC1:      "Cold HDD",
		VolumeTypeStandard: VolumeTypeMagnetic,
	}[volumeType]

	if volumeAPIName == "" {
		volumeAPIName = VolumeTypeGeneral
	}

	regionName := pricing.RegionMapping[region]
	if regionName == "" {
		regionName = region
	}

	return &pricing.PriceLookup{
		ServiceCode:   pricing.ServiceEC2,
		Region:        region,
		ProductFamily: "Storage",
		Attributes: map[string]string{
			"volumeApiName": volumeAPIName,
			"location":      regionName,
		},
	}, nil
}

func (h *EBSVolumeHandler) CalculateCost(price *pricing.Price, attrs map[string]interface{}) (hourly, monthly float64) {
	size := getFloatAttr(attrs, "size")
	if size == 0 {
		size = 8 // Default 8 GB
	}

	// EBS pricing is per GB-month
	monthly = price.OnDemandUSD * size

	// Add IOPS cost for io1/io2
	volumeType := getStringAttr(attrs, "type")
	if volumeType == VolumeTypeIO1 || volumeType == VolumeTypeIO2 {
		iops := getFloatAttr(attrs, "iops")
		if iops > 0 {
			// IOPS pricing would need separate lookup
			// For now, estimate $0.065 per provisioned IOPS-month (us-east-1 io1)
			monthly += iops * DefaultIO1IOPSCostPerMonth
		}
	}

	// Add throughput cost for gp3
	if volumeType == VolumeTypeGP3 {
		throughput := getFloatAttr(attrs, "throughput")
		if throughput > DefaultGP3FreeThroughputMBps {
			// $0.040 per MB/s-month over 125
			monthly += (throughput - DefaultGP3FreeThroughputMBps) * DefaultGP3ThroughputCostPerMB
		}
	}

	hourly = monthly / HoursPerMonth
	return hourly, monthly
}

// EIPHandler handles aws_eip cost estimation
type EIPHandler struct{}

func (h *EIPHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceEC2
}

func (h *EIPHandler) BuildLookup(region string, _ map[string]interface{}) (*pricing.PriceLookup, error) {
	regionName := pricing.RegionMapping[region]
	if regionName == "" {
		regionName = region
	}

	return &pricing.PriceLookup{
		ServiceCode:   pricing.ServiceEC2,
		Region:        region,
		ProductFamily: "IP Address",
		Attributes: map[string]string{
			"location": regionName,
			"group":    "ElasticIP:Address",
		},
	}, nil
}

func (h *EIPHandler) CalculateCost(price *pricing.Price, attrs map[string]interface{}) (hourly, monthly float64) {
	// EIP is free when attached to running instance
	// Cost is $0.005/hour when not attached (idle)
	// For estimation, assume it's attached (no cost) or idle
	instance := getStringAttr(attrs, "instance")
	if instance == "" {
		hourly = price.OnDemandUSD
		monthly = hourly * HoursPerMonth
	}
	return hourly, monthly
}

// NATGatewayHandler handles aws_nat_gateway cost estimation
type NATGatewayHandler struct{}

func (h *NATGatewayHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceEC2
}

func (h *NATGatewayHandler) BuildLookup(region string, _ map[string]interface{}) (*pricing.PriceLookup, error) {
	regionName := pricing.RegionMapping[region]
	if regionName == "" {
		regionName = region
	}

	return &pricing.PriceLookup{
		ServiceCode:   pricing.ServiceVPC,
		Region:        region,
		ProductFamily: "NAT Gateway",
		Attributes: map[string]string{
			"location": regionName,
			"group":    "NGW:NatGateway",
		},
	}, nil
}

func (h *NATGatewayHandler) CalculateCost(price *pricing.Price, _ map[string]interface{}) (hourly, monthly float64) {
	// NAT Gateway: hourly charge + data processing
	// For fixed cost estimation, only include hourly
	hourly = price.OnDemandUSD
	if hourly == 0 {
		hourly = DefaultNATGatewayHourlyCost
	}
	monthly = hourly * HoursPerMonth
	return hourly, monthly
}

// Helper functions

func getStringAttr(attrs map[string]interface{}, key string) string {
	if v, ok := attrs[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getFloatAttr(attrs map[string]interface{}, key string) float64 {
	if v, ok := attrs[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case int:
			return float64(val)
		case int64:
			return float64(val)
		}
	}
	return 0
}

func getIntAttr(attrs map[string]interface{}, key string) int {
	if v, ok := attrs[key]; ok {
		switch val := v.(type) {
		case float64:
			return int(val)
		case int:
			return val
		case int64:
			return int(val)
		}
	}
	return 0
}

func getBoolAttr(attrs map[string]interface{}, key string) bool {
	if v, ok := attrs[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}
