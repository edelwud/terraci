package aws

import (
	"github.com/edelwud/terraci/internal/cost/pricing"
)

// Serverless pricing constants
const (
	LambdaProvisionedConcurrencyCostPerGBSecond = 0.000004646
	LambdaDefaultMemoryMB                       = 128
	LambdaMemoryDivisor                         = 1024
	SecondsPerHour                              = 3600
	DynamoDBRCUCostPerHour                      = 0.00013
	DynamoDBWCUCostPerHour                      = 0.00065
	DynamoDBDefaultCapacity                     = 5
)

// LambdaHandler handles aws_lambda_function cost estimation
// Note: Lambda pricing is usage-based (requests + duration)
// For fixed cost estimation, we estimate based on memory and assume average invocations
type LambdaHandler struct{}

func (h *LambdaHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceLambda
}

func (h *LambdaHandler) BuildLookup(region string, _ map[string]interface{}) (*pricing.PriceLookup, error) {
	regionName := pricing.RegionMapping[region]
	if regionName == "" {
		regionName = region
	}

	return &pricing.PriceLookup{
		ServiceCode:   pricing.ServiceLambda,
		Region:        region,
		ProductFamily: "Serverless",
		Attributes: map[string]string{
			"location": regionName,
			"group":    "AWS-Lambda-Duration",
		},
	}, nil
}

func (h *LambdaHandler) CalculateCost(_ *pricing.Price, attrs map[string]interface{}) (hourly, monthly float64) {
	// Lambda has complex pricing: requests + GB-seconds
	// For fixed cost, return 0 as it's usage-based
	// Could estimate based on provisioned concurrency if set
	provisionedConcurrency := getIntAttr(attrs, "provisioned_concurrent_executions")
	if provisionedConcurrency > 0 {
		memoryMB := getIntAttr(attrs, "memory_size")
		if memoryMB == 0 {
			memoryMB = LambdaDefaultMemoryMB
		}
		// Provisioned concurrency: $0.000004646 per GB-second
		gbSeconds := float64(provisionedConcurrency) * (float64(memoryMB) / LambdaMemoryDivisor) * SecondsPerHour
		hourly = gbSeconds * LambdaProvisionedConcurrencyCostPerGBSecond
		monthly = hourly * HoursPerMonth
		return hourly, monthly
	}
	return 0, 0 // Usage-based, no fixed cost
}

// DynamoDBTableHandler handles aws_dynamodb_table cost estimation
type DynamoDBTableHandler struct{}

func (h *DynamoDBTableHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceDynamoDB
}

func (h *DynamoDBTableHandler) BuildLookup(region string, attrs map[string]interface{}) (*pricing.PriceLookup, error) {
	regionName := pricing.RegionMapping[region]
	if regionName == "" {
		regionName = region
	}

	// Check billing mode
	billingMode := getStringAttr(attrs, "billing_mode")
	if billingMode == "PAY_PER_REQUEST" {
		// On-demand: no lookup needed, usage-based
		return &pricing.PriceLookup{
			ServiceCode:   pricing.ServiceDynamoDB,
			Region:        region,
			ProductFamily: "Amazon DynamoDB PayPerRequest Throughput",
			Attributes: map[string]string{
				"location": regionName,
			},
		}, nil
	}

	// Provisioned: price per RCU/WCU
	return &pricing.PriceLookup{
		ServiceCode:   pricing.ServiceDynamoDB,
		Region:        region,
		ProductFamily: "Provisioned IOPS",
		Attributes: map[string]string{
			"location": regionName,
			"group":    "DDB-WriteUnits",
		},
	}, nil
}

func (h *DynamoDBTableHandler) CalculateCost(_ *pricing.Price, attrs map[string]interface{}) (hourly, monthly float64) {
	billingMode := getStringAttr(attrs, "billing_mode")
	if billingMode == "PAY_PER_REQUEST" {
		// On-demand: usage-based, no fixed cost
		return 0, 0
	}

	// Provisioned throughput
	readCapacity := getIntAttr(attrs, "read_capacity")
	writeCapacity := getIntAttr(attrs, "write_capacity")

	if readCapacity == 0 {
		readCapacity = DynamoDBDefaultCapacity
	}
	if writeCapacity == 0 {
		writeCapacity = DynamoDBDefaultCapacity
	}

	// Pricing varies by region, using us-east-1 defaults
	rcuCostPerHour := float64(readCapacity) * DynamoDBRCUCostPerHour
	wcuCostPerHour := float64(writeCapacity) * DynamoDBWCUCostPerHour

	hourly = rcuCostPerHour + wcuCostPerHour
	monthly = hourly * HoursPerMonth
	return hourly, monthly
}

// SQSQueueHandler handles aws_sqs_queue cost estimation
type SQSQueueHandler struct{}

func (h *SQSQueueHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceSQS
}

func (h *SQSQueueHandler) BuildLookup(_ string, _ map[string]interface{}) (*pricing.PriceLookup, error) {
	// SQS is usage-based (requests)
	return nil, nil
}

func (h *SQSQueueHandler) CalculateCost(_ *pricing.Price, _ map[string]interface{}) (hourly, monthly float64) {
	// SQS: $0.40 per million requests (first million free)
	// Usage-based, no fixed cost
	return 0, 0
}

// SNSTopicHandler handles aws_sns_topic cost estimation
type SNSTopicHandler struct{}

func (h *SNSTopicHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceSNS
}

func (h *SNSTopicHandler) BuildLookup(_ string, _ map[string]interface{}) (*pricing.PriceLookup, error) {
	// SNS is usage-based (publishes + deliveries)
	return nil, nil
}

func (h *SNSTopicHandler) CalculateCost(_ *pricing.Price, _ map[string]interface{}) (hourly, monthly float64) {
	// SNS: $0.50 per million requests
	// Usage-based, no fixed cost
	return 0, 0
}
