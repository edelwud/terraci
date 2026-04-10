package serverless

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// DynamoDB pricing constants
const (
	DynamoDBRCUCostPerHour  = 0.00013
	DynamoDBWCUCostPerHour  = 0.00065
	DynamoDBDefaultCapacity = 5
)

// DynamoDBHandler handles aws_dynamodb_table cost estimation
type DynamoDBHandler struct {
	awskit.RuntimeDeps
}

type dynamoDBAttrs struct {
	BillingMode   string
	ReadCapacity  int
	WriteCapacity int
}

func parseDynamoDBAttrs(attrs map[string]any) dynamoDBAttrs {
	return dynamoDBAttrs{
		BillingMode:   handler.GetStringAttr(attrs, "billing_mode"),
		ReadCapacity:  handler.GetIntAttr(attrs, "read_capacity"),
		WriteCapacity: handler.GetIntAttr(attrs, "write_capacity"),
	}
}

func (h *DynamoDBHandler) Category() handler.CostCategory { return handler.CostCategoryUsageBased }

func (h *DynamoDBHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	parsed := parseDynamoDBAttrs(attrs)
	if parsed.BillingMode == "PAY_PER_REQUEST" {
		spec := h.RuntimeOrDefault().StandardLookupSpec(
			awskit.ServiceKeyDynamoDB,
			"Amazon DynamoDB PayPerRequest Throughput",
			func(string, map[string]any) (map[string]string, error) {
				return nil, nil
			},
		)
		return spec.Build(region, attrs)
	}

	// Provisioned: price per RCU/WCU
	spec := h.RuntimeOrDefault().StandardLookupSpec(
		awskit.ServiceKeyDynamoDB,
		"Provisioned IOPS",
		func(string, map[string]any) (map[string]string, error) {
			return map[string]string{
				"group": "DDB-WriteUnits",
			}, nil
		},
	)
	return spec.Build(region, attrs)
}

func (h *DynamoDBHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	parsed := parseDynamoDBAttrs(attrs)
	return awskit.NewDescribeBuilder().
		String("billing_mode", parsed.BillingMode).
		Int("read_capacity", parsed.ReadCapacity).
		Int("write_capacity", parsed.WriteCapacity).
		Map()
}

func (h *DynamoDBHandler) CalculateUsageCost(_ string, attrs map[string]any) model.UsageCostEstimate {
	parsed := parseDynamoDBAttrs(attrs)
	if parsed.BillingMode == "PAY_PER_REQUEST" {
		// On-demand: usage-based, no fixed cost
		return model.UsageCostEstimate{Status: model.ResourceEstimateStatusUsageUnknown}
	}

	// Provisioned throughput
	readCapacity := parsed.ReadCapacity
	writeCapacity := parsed.WriteCapacity

	if readCapacity == 0 {
		readCapacity = DynamoDBDefaultCapacity
	}
	if writeCapacity == 0 {
		writeCapacity = DynamoDBDefaultCapacity
	}

	// Pricing varies by region, using us-east-1 defaults
	rcuCostPerHour := float64(readCapacity) * DynamoDBRCUCostPerHour
	wcuCostPerHour := float64(writeCapacity) * DynamoDBWCUCostPerHour

	hourly, monthly := handler.HourlyCost(rcuCostPerHour + wcuCostPerHour)
	return model.UsageCostEstimate{
		HourlyCost:  hourly,
		MonthlyCost: monthly,
		Status:      model.ResourceEstimateStatusUsageEstimated,
		Detail:      "usage-based estimate derived from provisioned throughput",
	}
}
