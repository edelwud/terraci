package aws

import "github.com/edelwud/terraci/plugins/cost/internal/pricing"

// HoursPerMonth is the average number of hours in a month for cost calculations.
const HoursPerMonth = 730

// ResolveRegionName returns the AWS pricing API region name for a region code.
// Falls back to the region code if no mapping exists.
func ResolveRegionName(region string) string {
	if name := pricing.RegionMapping[region]; name != "" {
		return name
	}
	return region
}

// RegionUsagePrefix maps AWS region codes to the pricing API usagetype prefix
// (e.g., "us-east-1" → "USE1"). Used for services like EKS and VPC that
// use region-prefixed usagetypes.
var RegionUsagePrefix = map[string]string{
	// US
	"us-east-1": "USE1",
	"us-east-2": "USE2",
	"us-west-1": "USW1",
	"us-west-2": "USW2",
	// Europe
	"eu-west-1":    "EUW1",
	"eu-west-2":    "EUW2",
	"eu-west-3":    "EUW3",
	"eu-central-1": "EUC1",
	"eu-central-2": "EUC2",
	"eu-north-1":   "EUN1",
	"eu-south-1":   "EUS1",
	"eu-south-2":   "EUS2",
	// Asia Pacific
	"ap-northeast-1": "APN1",
	"ap-northeast-2": "APN2",
	"ap-northeast-3": "APN3",
	"ap-southeast-1": "APS1",
	"ap-southeast-2": "APS2",
	"ap-southeast-3": "APS3",
	"ap-southeast-4": "APS4",
	"ap-south-1":     "APS5",
	"ap-south-2":     "APS6",
	"ap-east-1":      "APE1",
	// South America
	"sa-east-1": "SAE1",
	// Canada
	"ca-central-1": "CAN1",
	"ca-west-1":    "CAW1",
	// Middle East
	"me-south-1":   "MES1",
	"me-central-1": "MEC1",
	"il-central-1": "ILC1",
	// Africa
	"af-south-1": "AFS1",
}

// ResolveUsagePrefix returns the usagetype prefix for a region.
// Falls back to "USE1" (us-east-1) for unknown regions.
func ResolveUsagePrefix(region string) string {
	if prefix := RegionUsagePrefix[region]; prefix != "" {
		return prefix
	}
	return "USE1"
}
