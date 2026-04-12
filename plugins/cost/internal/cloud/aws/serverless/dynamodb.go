package serverless

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

// DynamoDB pricing constants
const (
	DynamoDBRCUCostPerHour  = 0.00013
	DynamoDBWCUCostPerHour  = 0.00065
	DynamoDBDefaultCapacity = 5
)

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

// DynamoDBSpec declares aws_dynamodb_table cost estimation.
func DynamoDBSpec(deps awskit.RuntimeDeps) resourcespec.ResourceSpec {
	return resourcespec.ResourceSpec{
		Type:     handler.ResourceType(awskit.ResourceDynamoDBTable),
		Category: handler.CostCategoryUsageBased,
		Lookup: &resourcespec.LookupSpec{
			BuildFunc: func(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
				parsed := parseDynamoDBAttrs(attrs)
				if parsed.BillingMode == "PAY_PER_REQUEST" {
					return deps.RuntimeOrDefault().StandardLookupSpec(
						awskit.ServiceKeyDynamoDB,
						"Amazon DynamoDB PayPerRequest Throughput",
						func(string, map[string]any) (map[string]string, error) { return nil, nil },
					).Build(region, attrs)
				}
				return deps.RuntimeOrDefault().StandardLookupSpec(
					awskit.ServiceKeyDynamoDB,
					"Provisioned IOPS",
					func(string, map[string]any) (map[string]string, error) {
						return map[string]string{"group": "DDB-WriteUnits"}, nil
					},
				).Build(region, attrs)
			},
		},
		Describe: &resourcespec.DescribeSpec{
			BuildFunc: func(_ *pricing.Price, attrs map[string]any) map[string]string {
				parsed := parseDynamoDBAttrs(attrs)
				return awskit.NewDescribeBuilder().
					String("billing_mode", parsed.BillingMode).
					Int("read_capacity", parsed.ReadCapacity).
					Int("write_capacity", parsed.WriteCapacity).
					Map()
			},
		},
		Usage: &resourcespec.UsagePricingSpec{
			EstimateFunc: func(_ string, attrs map[string]any) model.UsageCostEstimate {
				parsed := parseDynamoDBAttrs(attrs)
				if parsed.BillingMode == "PAY_PER_REQUEST" {
					return model.UsageCostEstimate{Status: model.ResourceEstimateStatusUsageUnknown}
				}

				readCapacity := parsed.ReadCapacity
				writeCapacity := parsed.WriteCapacity
				if readCapacity == 0 {
					readCapacity = DynamoDBDefaultCapacity
				}
				if writeCapacity == 0 {
					writeCapacity = DynamoDBDefaultCapacity
				}

				rcuCostPerHour := float64(readCapacity) * DynamoDBRCUCostPerHour
				wcuCostPerHour := float64(writeCapacity) * DynamoDBWCUCostPerHour
				hourly, monthly := handler.HourlyCost(rcuCostPerHour + wcuCostPerHour)
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
