package storage

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

// CloudWatch pricing constants.
const (
	CloudWatchStandardAlarmCost    = 0.10
	CloudWatchHighResAlarmCost     = 0.30
	HighResolutionThresholdSeconds = 60
)

// LogGroupSpec declares aws_cloudwatch_log_group cost estimation.
func LogGroupSpec() resourcespec.ResourceSpec {
	return resourcespec.ResourceSpec{
		Type:     resourcedef.ResourceType(awskit.ResourceCloudWatchLogGroup),
		Category: resourcedef.CostCategoryUsageBased,
		Usage: &resourcespec.UsagePricingSpec{
			EstimateFunc: func(_ string, _ map[string]any) model.UsageCostEstimate {
				return model.UsageCostEstimate{Status: model.ResourceEstimateStatusUsageUnknown}
			},
		},
	}
}

type alarmAttrs struct {
	Period int
}

func parseAlarmAttrs(attrs map[string]any) alarmAttrs {
	return alarmAttrs{
		Period: costutil.GetIntAttr(attrs, "period"),
	}
}

// AlarmSpec declares aws_cloudwatch_metric_alarm cost estimation.
func AlarmSpec() resourcespec.ResourceSpec {
	return resourcespec.ResourceSpec{
		Type:     resourcedef.ResourceType(awskit.ResourceCloudWatchMetricAlarm),
		Category: resourcedef.CostCategoryFixed,
		Lookup: &resourcespec.LookupSpec{
			BuildFunc: func(_ string, _ map[string]any) (*pricing.PriceLookup, error) { return nil, nil },
		},
		Describe: &resourcespec.DescribeSpec{
			BuildFunc: func(_ *pricing.Price, attrs map[string]any) map[string]string {
				desc := make(map[string]string)
				parsed := parseAlarmAttrs(attrs)
				if parsed.Period > 0 && parsed.Period < HighResolutionThresholdSeconds {
					desc["resolution"] = "high"
				} else {
					desc["resolution"] = "standard"
				}
				return desc
			},
		},
		Fixed: &resourcespec.FixedPricingSpec{
			CostFunc: func(_ string, attrs map[string]any) (hourly, monthly float64) {
				parsed := parseAlarmAttrs(attrs)
				if parsed.Period > 0 && parsed.Period < HighResolutionThresholdSeconds {
					return costutil.FixedMonthlyCost(CloudWatchHighResAlarmCost)
				}
				return costutil.FixedMonthlyCost(CloudWatchStandardAlarmCost)
			},
		},
	}
}
