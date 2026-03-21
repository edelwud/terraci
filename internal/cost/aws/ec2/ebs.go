package ec2

import (
	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

// EBS pricing constants
const (
	DefaultIO1IOPSCostPerMonth    = 0.065
	DefaultGP3ThroughputCostPerMB = 0.040
	DefaultGP3FreeThroughputMBps  = 125
)

// EBSHandler handles aws_ebs_volume cost estimation.
type EBSHandler struct{}

func (h *EBSHandler) Category() aws.CostCategory { return aws.CostCategoryStandard }

func (h *EBSHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceEC2
}

func (h *EBSHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	volumeType := aws.GetStringAttr(attrs, "type")
	if volumeType == "" {
		volumeType = aws.VolumeTypeGP2 // Default
	}

	// Map terraform volume types to AWS pricing attributes
	volumeAPIName := map[string]string{
		aws.VolumeTypeGP2:      aws.VolumeTypeGeneral,
		aws.VolumeTypeGP3:      aws.VolumeTypeGeneral,
		aws.VolumeTypeIO1:      aws.VolumeTypeProvisioned,
		aws.VolumeTypeIO2:      aws.VolumeTypeProvisioned,
		aws.VolumeTypeST1:      "Throughput Optimized HDD",
		aws.VolumeTypeSC1:      "Cold HDD",
		aws.VolumeTypeStandard: aws.VolumeTypeMagnetic,
	}[volumeType]

	if volumeAPIName == "" {
		volumeAPIName = aws.VolumeTypeGeneral
	}

	lb := &aws.LookupBuilder{Service: pricing.ServiceEC2, ProductFamily: "Storage"}
	return lb.Build(region, map[string]string{
		"volumeApiName": volumeAPIName,
	}), nil
}

func (h *EBSHandler) CalculateCost(price *pricing.Price, attrs map[string]any) (hourly, monthly float64) {
	size := aws.GetFloatAttr(attrs, "size")
	if size == 0 {
		size = 8 // Default 8 GB
	}

	// EBS pricing is per GB-month
	monthly = price.OnDemandUSD * size

	// Add IOPS cost for io1/io2
	volumeType := aws.GetStringAttr(attrs, "type")
	if volumeType == aws.VolumeTypeIO1 || volumeType == aws.VolumeTypeIO2 {
		iops := aws.GetFloatAttr(attrs, "iops")
		if iops > 0 {
			// IOPS pricing would need separate lookup
			// For now, estimate $0.065 per provisioned IOPS-month (us-east-1 io1)
			monthly += iops * DefaultIO1IOPSCostPerMonth
		}
	}

	// Add throughput cost for gp3
	if volumeType == aws.VolumeTypeGP3 {
		throughput := aws.GetFloatAttr(attrs, "throughput")
		if throughput > DefaultGP3FreeThroughputMBps {
			// $0.040 per MB/s-month over 125
			monthly += (throughput - DefaultGP3FreeThroughputMBps) * DefaultGP3ThroughputCostPerMB
		}
	}

	hourly = monthly / aws.HoursPerMonth
	return hourly, monthly
}
