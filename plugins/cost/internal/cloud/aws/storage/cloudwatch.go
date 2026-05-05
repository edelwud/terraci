package storage

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

// CloudWatch pricing constants.
const (
	CloudWatchStandardAlarmCost    = 0.10
	CloudWatchHighResAlarmCost     = 0.30
	HighResolutionThresholdSeconds = 60

	// alarmResolutionHigh / alarmResolutionStandard are the resolution
	// labels surfaced in the Describe payload and asserted in tests.
	alarmResolutionHigh     = "high"
	alarmResolutionStandard = "standard"
)

// LogGroupSpec declares aws_cloudwatch_log_group cost estimation.
func LogGroupSpec() resourcespec.TypedSpec[resourcespec.NoAttrs] {
	return resourcespec.UsageUnknownNoAttrsSpec(resourcedef.ResourceType(awskit.ResourceCloudWatchLogGroup))
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
func AlarmSpec() resourcespec.TypedSpec[alarmAttrs] {
	return resourcespec.TypedSpec[alarmAttrs]{
		Type:     resourcedef.ResourceType(awskit.ResourceCloudWatchMetricAlarm),
		Category: resourcedef.CostCategoryFixed,
		Parse:    parseAlarmAttrs,
		Describe: &resourcespec.TypedDescribeSpec[alarmAttrs]{
			BuildFunc: func(_ *pricing.Price, p alarmAttrs) map[string]string {
				return awskit.NewDescribeBuilder().
					String("resolution", map[bool]string{true: alarmResolutionHigh, false: alarmResolutionStandard}[p.Period > 0 && p.Period < HighResolutionThresholdSeconds]).
					Map()
			},
		},
		Fixed: &resourcespec.TypedFixedPricingSpec[alarmAttrs]{
			CostFunc: func(_ string, p alarmAttrs) (hourly, monthly float64) {
				if p.Period > 0 && p.Period < HighResolutionThresholdSeconds {
					return costutil.FixedMonthlyCost(CloudWatchHighResAlarmCost)
				}
				return costutil.FixedMonthlyCost(CloudWatchStandardAlarmCost)
			},
		},
	}
}
