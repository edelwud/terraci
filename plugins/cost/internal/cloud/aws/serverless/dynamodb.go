package serverless

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

// DynamoDB pricing constants
const (
	DynamoDBRCUCostPerHour  = 0.00013
	DynamoDBWCUCostPerHour  = 0.00065
	DynamoDBDefaultCapacity = 5

	// DynamoDB billing modes (terraform attribute values).
	billingModePayPerRequest = "PAY_PER_REQUEST"
)

type dynamoDBAttrs struct {
	BillingMode   string
	ReadCapacity  int
	WriteCapacity int
}

func parseDynamoDBAttrs(attrs map[string]any) dynamoDBAttrs {
	return dynamoDBAttrs{
		BillingMode:   costutil.GetStringAttr(attrs, "billing_mode"),
		ReadCapacity:  costutil.GetIntAttr(attrs, "read_capacity"),
		WriteCapacity: costutil.GetIntAttr(attrs, "write_capacity"),
	}
}

// DynamoDBSpec declares aws_dynamodb_table cost estimation.
func DynamoDBSpec(deps awskit.RuntimeDeps) resourcespec.TypedSpec[dynamoDBAttrs] {
	return resourcespec.TypedSpec[dynamoDBAttrs]{
		Type:     resourcedef.ResourceType(awskit.ResourceDynamoDBTable),
		Category: resourcedef.CostCategoryUsageBased,
		Parse:    parseDynamoDBAttrs,
		Lookup: &resourcespec.TypedLookupSpec[dynamoDBAttrs]{
			BuildFunc: func(region string, p dynamoDBAttrs) (*pricing.PriceLookup, error) {
				runtime := deps.RuntimeOrDefault()
				if p.BillingMode == billingModePayPerRequest {
					return runtime.
						NewLookupBuilder(awskit.ServiceKeyDynamoDB, "Amazon DynamoDB PayPerRequest Throughput").
						Build(region), nil
				}
				return runtime.
					NewLookupBuilder(awskit.ServiceKeyDynamoDB, "Provisioned IOPS").
					Attr("group", "DDB-WriteUnits").
					Build(region), nil
			},
		},
		Describe: &resourcespec.TypedDescribeSpec[dynamoDBAttrs]{
			BuildFunc: func(_ *pricing.Price, p dynamoDBAttrs) map[string]string {
				return awskit.NewDescribeBuilder().
					String("billing_mode", p.BillingMode).
					Int("read_capacity", p.ReadCapacity).
					Int("write_capacity", p.WriteCapacity).
					Map()
			},
		},
		Usage: &resourcespec.TypedUsagePricingSpec[dynamoDBAttrs]{
			EstimateFunc: func(_ string, p dynamoDBAttrs) model.UsageCostEstimate {
				if p.BillingMode == billingModePayPerRequest {
					return model.UsageCostEstimate{Status: model.ResourceEstimateStatusUsageUnknown}
				}

				readCapacity := p.ReadCapacity
				writeCapacity := p.WriteCapacity
				if readCapacity == 0 {
					readCapacity = DynamoDBDefaultCapacity
				}
				if writeCapacity == 0 {
					writeCapacity = DynamoDBDefaultCapacity
				}

				rcuCostPerHour := float64(readCapacity) * DynamoDBRCUCostPerHour
				wcuCostPerHour := float64(writeCapacity) * DynamoDBWCUCostPerHour
				hourly, monthly := costutil.HourlyCost(rcuCostPerHour + wcuCostPerHour)
				return model.UsageCostEstimate{
					HourlyCost:  hourly,
					MonthlyCost: monthly,
					Status:      model.ResourceEstimateStatusUsageEstimated,
					Detail:      "usage-based estimate derived from provisioned throughput",
				}
			},
		},
	}
}
