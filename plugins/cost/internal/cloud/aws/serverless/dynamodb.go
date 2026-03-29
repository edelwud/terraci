package serverless

import (
	"strconv"

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
type DynamoDBHandler struct{}

func (h *DynamoDBHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *DynamoDBHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceDynamoDB
}

func (h *DynamoDBHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	// Check billing mode
	billingMode := handler.GetStringAttr(attrs, "billing_mode")
	if billingMode == "PAY_PER_REQUEST" {
		// On-demand: no lookup needed, usage-based
		lb := &awskit.LookupBuilder{Service: pricing.ServiceDynamoDB, ProductFamily: "Amazon DynamoDB PayPerRequest Throughput"}
		return lb.Build(region, nil), nil
	}

	// Provisioned: price per RCU/WCU
	lb := &awskit.LookupBuilder{Service: pricing.ServiceDynamoDB, ProductFamily: "Provisioned IOPS"}
	return lb.Build(region, map[string]string{
		"group": "DDB-WriteUnits",
	}), nil
}

func (h *DynamoDBHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	desc := make(map[string]string)
	if v := handler.GetStringAttr(attrs, "billing_mode"); v != "" {
		desc["billing_mode"] = v
	}
	if v := handler.GetIntAttr(attrs, "read_capacity"); v != 0 {
		desc["read_capacity"] = strconv.Itoa(v)
	}
	if v := handler.GetIntAttr(attrs, "write_capacity"); v != 0 {
		desc["write_capacity"] = strconv.Itoa(v)
	}
	return desc
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
