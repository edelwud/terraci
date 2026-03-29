package awskit

import "github.com/edelwud/terraci/plugins/cost/internal/pricing"

// awsRegionMapping maps AWS region codes to pricing API region names.
var awsRegionMapping = map[string]string{
	// US
	"us-east-1": "US East (N. Virginia)",
	"us-east-2": "US East (Ohio)",
	"us-west-1": "US West (N. California)",
	"us-west-2": "US West (Oregon)",
	// Europe
	"eu-west-1":    "EU (Ireland)",
	"eu-west-2":    "EU (London)",
	"eu-west-3":    "EU (Paris)",
	"eu-central-1": "EU (Frankfurt)",
	"eu-central-2": "EU (Zurich)",
	"eu-north-1":   "EU (Stockholm)",
	"eu-south-1":   "EU (Milan)",
	"eu-south-2":   "EU (Spain)",
	// Asia Pacific
	"ap-northeast-1": "Asia Pacific (Tokyo)",
	"ap-northeast-2": "Asia Pacific (Seoul)",
	"ap-northeast-3": "Asia Pacific (Osaka)",
	"ap-southeast-1": "Asia Pacific (Singapore)",
	"ap-southeast-2": "Asia Pacific (Sydney)",
	"ap-southeast-3": "Asia Pacific (Jakarta)",
	"ap-southeast-4": "Asia Pacific (Melbourne)",
	"ap-south-1":     "Asia Pacific (Mumbai)",
	"ap-south-2":     "Asia Pacific (Hyderabad)",
	"ap-east-1":      "Asia Pacific (Hong Kong)",
	// South America
	"sa-east-1": "South America (Sao Paulo)",
	// Canada
	"ca-central-1": "Canada (Central)",
	"ca-west-1":    "Canada West (Calgary)",
	// Middle East
	"me-south-1":   "Middle East (Bahrain)",
	"me-central-1": "Middle East (UAE)",
	"il-central-1": "Israel (Tel Aviv)",
	// Africa
	"af-south-1": "Africa (Cape Town)",
}

// InitRegionMapping registers AWS region mapping with the pricing package.
// Called automatically by NewRegistry.
func InitRegionMapping() {
	pricing.SetRegionMapping(awsRegionMapping)
}

// ResolveRegionName returns the AWS pricing API region name for a region code.
// Falls back to the region code if no mapping exists.
func ResolveRegionName(region string) string {
	if name := awsRegionMapping[region]; name != "" {
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

// DefaultUsagePrefix is the fallback usagetype prefix (us-east-1).
const DefaultUsagePrefix = "USE1"

// ResolveUsagePrefix returns the usagetype prefix for a region.
// Falls back to DefaultUsagePrefix (us-east-1) for unknown regions.
func ResolveUsagePrefix(region string) string {
	if prefix := RegionUsagePrefix[region]; prefix != "" {
		return prefix
	}
	return DefaultUsagePrefix
}
