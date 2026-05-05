package awskit

// AWS region code constants. Used as keys in the mapping tables below and
// recommended for callers that need region literals — keeps spelling
// uniform and quiets goconst about repeated string literals.
const (
	RegionUSEast1      = "us-east-1"
	RegionUSEast2      = "us-east-2"
	RegionUSWest1      = "us-west-1"
	RegionUSWest2      = "us-west-2"
	RegionEUWest1      = "eu-west-1"
	RegionEUWest2      = "eu-west-2"
	RegionEUWest3      = "eu-west-3"
	RegionEUCentral1   = "eu-central-1"
	RegionEUCentral2   = "eu-central-2"
	RegionEUNorth1     = "eu-north-1"
	RegionEUSouth1     = "eu-south-1"
	RegionEUSouth2     = "eu-south-2"
	RegionAPNortheast1 = "ap-northeast-1"
	RegionAPNortheast2 = "ap-northeast-2"
	RegionAPNortheast3 = "ap-northeast-3"
	RegionAPSoutheast1 = "ap-southeast-1"
	RegionAPSoutheast2 = "ap-southeast-2"
	RegionAPSoutheast3 = "ap-southeast-3"
	RegionAPSoutheast4 = "ap-southeast-4"
	RegionAPSouth1     = "ap-south-1"
	RegionAPSouth2     = "ap-south-2"
	RegionAPEast1      = "ap-east-1"
	RegionSAEast1      = "sa-east-1"
	RegionCACentral1   = "ca-central-1"
	RegionCAWest1      = "ca-west-1"
	RegionMESouth1     = "me-south-1"
	RegionMECentral1   = "me-central-1"
	RegionILCentral1   = "il-central-1"
	RegionAFSouth1     = "af-south-1"
)

// AWS pricing-API region display names that recur across the codebase
// (mostly in mapping tables and tests). Promoted to constants to satisfy
// goconst — values are unchanged.
const (
	regionNameEUWest1 = "EU (Ireland)"
)

// AWS pricing-API region usagetype prefixes that recur across the codebase.
const (
	// DefaultUsagePrefix is the fallback usagetype prefix (us-east-1).
	DefaultUsagePrefix = "USE1"
	usagePrefixEUWest1 = "EUW1"
)

// awsRegionMapping maps AWS region codes to pricing API region names.
var awsRegionMapping = map[string]string{
	// US
	RegionUSEast1: "US East (N. Virginia)",
	RegionUSEast2: "US East (Ohio)",
	RegionUSWest1: "US West (N. California)",
	RegionUSWest2: "US West (Oregon)",
	// Europe
	RegionEUWest1:    regionNameEUWest1,
	RegionEUWest2:    "EU (London)",
	RegionEUWest3:    "EU (Paris)",
	RegionEUCentral1: "EU (Frankfurt)",
	RegionEUCentral2: "EU (Zurich)",
	RegionEUNorth1:   "EU (Stockholm)",
	RegionEUSouth1:   "EU (Milan)",
	RegionEUSouth2:   "EU (Spain)",
	// Asia Pacific
	RegionAPNortheast1: "Asia Pacific (Tokyo)",
	RegionAPNortheast2: "Asia Pacific (Seoul)",
	RegionAPNortheast3: "Asia Pacific (Osaka)",
	RegionAPSoutheast1: "Asia Pacific (Singapore)",
	RegionAPSoutheast2: "Asia Pacific (Sydney)",
	RegionAPSoutheast3: "Asia Pacific (Jakarta)",
	RegionAPSoutheast4: "Asia Pacific (Melbourne)",
	RegionAPSouth1:     "Asia Pacific (Mumbai)",
	RegionAPSouth2:     "Asia Pacific (Hyderabad)",
	RegionAPEast1:      "Asia Pacific (Hong Kong)",
	// South America
	RegionSAEast1: "South America (Sao Paulo)",
	// Canada
	RegionCACentral1: "Canada (Central)",
	RegionCAWest1:    "Canada West (Calgary)",
	// Middle East
	RegionMESouth1:   "Middle East (Bahrain)",
	RegionMECentral1: "Middle East (UAE)",
	RegionILCentral1: "Israel (Tel Aviv)",
	// Africa
	RegionAFSouth1: "Africa (Cape Town)",
}

// awsRegionUsagePrefix maps AWS region codes to the pricing API usagetype
// prefix (e.g., "us-east-1" → "USE1"). Used for services like EKS and VPC
// that use region-prefixed usagetypes.
var awsRegionUsagePrefix = map[string]string{
	// US
	RegionUSEast1: DefaultUsagePrefix,
	RegionUSEast2: "USE2",
	RegionUSWest1: "USW1",
	RegionUSWest2: "USW2",
	// Europe
	RegionEUWest1:    usagePrefixEUWest1,
	RegionEUWest2:    "EUW2",
	RegionEUWest3:    "EUW3",
	RegionEUCentral1: "EUC1",
	RegionEUCentral2: "EUC2",
	RegionEUNorth1:   "EUN1",
	RegionEUSouth1:   "EUS1",
	RegionEUSouth2:   "EUS2",
	// Asia Pacific
	RegionAPNortheast1: "APN1",
	RegionAPNortheast2: "APN2",
	RegionAPNortheast3: "APN3",
	RegionAPSoutheast1: "APS1",
	RegionAPSoutheast2: "APS2",
	RegionAPSoutheast3: "APS3",
	RegionAPSoutheast4: "APS4",
	RegionAPSouth1:     "APS5",
	RegionAPSouth2:     "APS6",
	RegionAPEast1:      "APE1",
	// South America
	RegionSAEast1: "SAE1",
	// Canada
	RegionCACentral1: "CAN1",
	RegionCAWest1:    "CAW1",
	// Middle East
	RegionMESouth1:   "MES1",
	RegionMECentral1: "MEC1",
	RegionILCentral1: "ILC1",
	// Africa
	RegionAFSouth1: "AFS1",
}
