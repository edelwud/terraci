package serverless

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
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

func (h *DynamoDBHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *DynamoDBHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	// Check billing mode
	billingMode := handler.GetStringAttr(attrs, "billing_mode")
	if billingMode == "PAY_PER_REQUEST" {
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
	return awskit.NewDescribeBuilder().
		String("billing_mode", handler.GetStringAttr(attrs, "billing_mode")).
		Int("read_capacity", handler.GetIntAttr(attrs, "read_capacity")).
		Int("write_capacity", handler.GetIntAttr(attrs, "write_capacity")).
		Map()
}

func (h *DynamoDBHandler) CalculateCost(_ *pricing.Price, _ *pricing.PriceIndex, _ string, attrs map[string]any) (hourly, monthly float64) {
	billingMode := handler.GetStringAttr(attrs, "billing_mode")
	if billingMode == "PAY_PER_REQUEST" {
		// On-demand: usage-based, no fixed cost
		return 0, 0
	}

	// Provisioned throughput
	readCapacity := handler.GetIntAttr(attrs, "read_capacity")
	writeCapacity := handler.GetIntAttr(attrs, "write_capacity")

	if readCapacity == 0 {
		readCapacity = DynamoDBDefaultCapacity
	}
	if writeCapacity == 0 {
		writeCapacity = DynamoDBDefaultCapacity
	}

	// Pricing varies by region, using us-east-1 defaults
	rcuCostPerHour := float64(readCapacity) * DynamoDBRCUCostPerHour
	wcuCostPerHour := float64(writeCapacity) * DynamoDBWCUCostPerHour

	return handler.HourlyCost(rcuCostPerHour + wcuCostPerHour)
}
