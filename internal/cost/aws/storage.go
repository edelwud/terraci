package aws

import (
	"github.com/edelwud/terraci/internal/cost/pricing"
)

// Storage pricing constants
const (
	CloudWatchStandardAlarmCost    = 0.10
	CloudWatchHighResAlarmCost     = 0.30
	SecretsManagerSecretCost       = 0.40
	KMSKeyCost                     = 1.00
	Route53HostedZoneCost          = 0.50
	HighResolutionThresholdSeconds = 60
)

// S3BucketHandler handles aws_s3_bucket cost estimation
// Note: S3 is primarily usage-based (storage + requests)
// For fixed cost, we can't estimate without usage data
type S3BucketHandler struct{}

func (h *S3BucketHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceS3
}

func (h *S3BucketHandler) BuildLookup(_ string, _ map[string]interface{}) (*pricing.PriceLookup, error) {
	// S3 bucket itself is free, cost is for storage and requests
	return nil, nil
}

func (h *S3BucketHandler) CalculateCost(_ *pricing.Price, _ map[string]interface{}) (hourly, monthly float64) {
	// S3: ~$0.023 per GB-month for Standard
	// Without usage data, we can't estimate
	return 0, 0
}

// CloudWatchLogGroupHandler handles aws_cloudwatch_log_group cost estimation
type CloudWatchLogGroupHandler struct{}

func (h *CloudWatchLogGroupHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceCloudWatch
}

func (h *CloudWatchLogGroupHandler) BuildLookup(_ string, _ map[string]interface{}) (*pricing.PriceLookup, error) {
	// CloudWatch Logs: ingestion + storage
	return nil, nil
}

func (h *CloudWatchLogGroupHandler) CalculateCost(_ *pricing.Price, _ map[string]interface{}) (hourly, monthly float64) {
	// CloudWatch Logs: $0.50 per GB ingested, $0.03 per GB stored
	// Usage-based, no fixed cost
	return 0, 0
}

// CloudWatchAlarmHandler handles aws_cloudwatch_metric_alarm cost estimation
type CloudWatchAlarmHandler struct{}

func (h *CloudWatchAlarmHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceCloudWatch
}

func (h *CloudWatchAlarmHandler) BuildLookup(region string, _ map[string]interface{}) (*pricing.PriceLookup, error) {
	regionName := pricing.RegionMapping[region]
	if regionName == "" {
		regionName = region
	}

	return &pricing.PriceLookup{
		ServiceCode:   pricing.ServiceCloudWatch,
		Region:        region,
		ProductFamily: "Alarm",
		Attributes: map[string]string{
			"location": regionName,
		},
	}, nil
}

func (h *CloudWatchAlarmHandler) CalculateCost(_ *pricing.Price, attrs map[string]interface{}) (hourly, monthly float64) {
	// Standard resolution alarm: $0.10/alarm/month
	// High resolution alarm: $0.30/alarm/month
	period := getIntAttr(attrs, "period")
	if period > 0 && period < HighResolutionThresholdSeconds {
		monthly = CloudWatchHighResAlarmCost
	} else {
		monthly = CloudWatchStandardAlarmCost
	}
	hourly = monthly / HoursPerMonth
	return hourly, monthly
}

// SecretsManagerHandler handles aws_secretsmanager_secret cost estimation
type SecretsManagerHandler struct{}

func (h *SecretsManagerHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceSecretsMan
}

func (h *SecretsManagerHandler) BuildLookup(region string, _ map[string]interface{}) (*pricing.PriceLookup, error) {
	regionName := pricing.RegionMapping[region]
	if regionName == "" {
		regionName = region
	}

	return &pricing.PriceLookup{
		ServiceCode:   pricing.ServiceSecretsMan,
		Region:        region,
		ProductFamily: "Secret",
		Attributes: map[string]string{
			"location": regionName,
		},
	}, nil
}

func (h *SecretsManagerHandler) CalculateCost(_ *pricing.Price, _ map[string]interface{}) (hourly, monthly float64) {
	// Secrets Manager: $0.40 per secret per month + $0.05 per 10,000 API calls
	monthly = SecretsManagerSecretCost
	hourly = monthly / HoursPerMonth
	return hourly, monthly
}

// KMSKeyHandler handles aws_kms_key cost estimation
type KMSKeyHandler struct{}

func (h *KMSKeyHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceKMS
}

func (h *KMSKeyHandler) BuildLookup(region string, _ map[string]interface{}) (*pricing.PriceLookup, error) {
	regionName := pricing.RegionMapping[region]
	if regionName == "" {
		regionName = region
	}

	return &pricing.PriceLookup{
		ServiceCode:   pricing.ServiceKMS,
		Region:        region,
		ProductFamily: "Key Management Service",
		Attributes: map[string]string{
			"location": regionName,
		},
	}, nil
}

func (h *KMSKeyHandler) CalculateCost(_ *pricing.Price, _ map[string]interface{}) (hourly, monthly float64) {
	// KMS: $1.00 per key per month (customer managed keys)
	// AWS managed keys are free
	// All key types have the same base cost
	monthly = KMSKeyCost
	hourly = monthly / HoursPerMonth
	return hourly, monthly
}

// Route53ZoneHandler handles aws_route53_zone cost estimation
type Route53ZoneHandler struct{}

func (h *Route53ZoneHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceRoute53
}

func (h *Route53ZoneHandler) BuildLookup(_ string, _ map[string]interface{}) (*pricing.PriceLookup, error) {
	return &pricing.PriceLookup{
		ServiceCode:   pricing.ServiceRoute53,
		Region:        "global",
		ProductFamily: "DNS Zone",
		Attributes:    map[string]string{},
	}, nil
}

func (h *Route53ZoneHandler) CalculateCost(_ *pricing.Price, _ map[string]interface{}) (hourly, monthly float64) {
	// Route53: $0.50 per hosted zone per month (first 25 zones)
	// Then $0.10 per zone after 25
	monthly = Route53HostedZoneCost
	hourly = monthly / HoursPerMonth
	return hourly, monthly
}
