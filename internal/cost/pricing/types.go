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

// RegionMapping maps AWS region codes to pricing region names
var RegionMapping = map[string]string{
	"us-east-1":      "US East (N. Virginia)",
	"us-east-2":      "US East (Ohio)",
	"us-west-1":      "US West (N. California)",
	"us-west-2":      "US West (Oregon)",
	"eu-west-1":      "EU (Ireland)",
	"eu-west-2":      "EU (London)",
	"eu-west-3":      "EU (Paris)",
	"eu-central-1":   "EU (Frankfurt)",
	"eu-north-1":     "EU (Stockholm)",
	"eu-south-1":     "EU (Milan)",
	"ap-northeast-1": "Asia Pacific (Tokyo)",
	"ap-northeast-2": "Asia Pacific (Seoul)",
	"ap-northeast-3": "Asia Pacific (Osaka)",
	"ap-southeast-1": "Asia Pacific (Singapore)",
	"ap-southeast-2": "Asia Pacific (Sydney)",
	"ap-south-1":     "Asia Pacific (Mumbai)",
	"sa-east-1":      "South America (Sao Paulo)",
	"ca-central-1":   "Canada (Central)",
	"me-south-1":     "Middle East (Bahrain)",
	"af-south-1":     "Africa (Cape Town)",
}

// RegionCodeMapping is reverse mapping from pricing region name to code
var RegionCodeMapping = func() map[string]string {
	m := make(map[string]string)
	for code, name := range RegionMapping {
		m[name] = code
	}
	return m
}()
