// Package pricing provides AWS Price List API integration
package pricing

import "time"

// ServiceCode represents an AWS service code for pricing API
type ServiceCode string

// AWS service codes for pricing
const (
	ServiceEC2         ServiceCode = "AmazonEC2"
	ServiceRDS         ServiceCode = "AmazonRDS"
	ServiceS3          ServiceCode = "AmazonS3"
	ServiceEBS         ServiceCode = "AmazonEC2" // EBS is part of EC2 pricing
	ServiceELB         ServiceCode = "AWSELB"
	ServiceELBv2       ServiceCode = "AmazonEC2" // ALB/NLB pricing in EC2
	ServiceLambda      ServiceCode = "AWSLambda"
	ServiceDynamoDB    ServiceCode = "AmazonDynamoDB"
	ServiceCloudWatch  ServiceCode = "AmazonCloudWatch"
	ServiceSNS         ServiceCode = "AmazonSNS"
	ServiceSQS         ServiceCode = "AWSQueueService"
	ServiceElastiCache ServiceCode = "AmazonElastiCache"
	ServiceEKS         ServiceCode = "AmazonEKS"
	ServiceECS         ServiceCode = "AmazonECS"
	ServiceSecretsMan  ServiceCode = "AWSSecretsManager"
	ServiceKMS         ServiceCode = "awskms"
	ServiceRoute53     ServiceCode = "AmazonRoute53"
	ServiceCloudFront  ServiceCode = "AmazonCloudFront"
	ServiceNAT         ServiceCode = "AmazonEC2" // NAT Gateway in EC2
	ServiceVPC         ServiceCode = "AmazonVPC"
)

// PriceIndex represents a compact pricing index for a service/region
type PriceIndex struct {
	ServiceCode ServiceCode       `json:"service_code"`
	Region      string            `json:"region"`
	Version     string            `json:"version"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Products    map[string]Price  `json:"products"` // SKU -> Price
	Attributes  map[string]string `json:"attributes,omitempty"`
}

// Price represents a single product price
type Price struct {
	SKU           string            `json:"sku"`
	ProductFamily string            `json:"product_family"`
	Attributes    map[string]string `json:"attributes"`
	OnDemandUSD   float64           `json:"on_demand_usd"` // OnDemand hourly price in USD
	Unit          string            `json:"unit"`          // Hrs, GB-Mo, etc.
}

// PriceLookup represents criteria for finding a price
type PriceLookup struct {
	ServiceCode   ServiceCode
	Region        string
	ProductFamily string
	Attributes    map[string]string
}

// isValid checks if the index contains usable data.
func (idx *PriceIndex) isValid() bool {
	return idx != nil && idx.ServiceCode != "" && idx.Region != "" && len(idx.Products) > 0
}

// RegionMapping is set by provider packages (e.g., aws/) to map region codes to pricing names.
// Provider-agnostic: each provider populates this with its own region mapping.
var RegionMapping map[string]string

// RegionCodeMapping is the reverse mapping from pricing region name to code.
// Rebuilt when SetRegionMapping is called.
var RegionCodeMapping map[string]string

// SetRegionMapping sets the region mapping and rebuilds the reverse mapping.
// Called by provider packages during init.
func SetRegionMapping(m map[string]string) {
	RegionMapping = m
	reverse := make(map[string]string, len(m))
	for code, name := range m {
		reverse[name] = code
	}
	RegionCodeMapping = reverse
}
