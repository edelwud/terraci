package storage

import (
	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

// CloudWatch pricing constants
const (
	CloudWatchStandardAlarmCost    = 0.10
	CloudWatchHighResAlarmCost     = 0.30
	HighResolutionThresholdSeconds = 60
)

// LogGroupHandler handles aws_cloudwatch_log_group cost estimation
type LogGroupHandler struct{}

func (h *LogGroupHandler) Category() aws.CostCategory { return aws.CostCategoryUsageBased }

func (h *LogGroupHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceCloudWatch
}

func (h *LogGroupHandler) BuildLookup(_ string, _ map[string]any) (*pricing.PriceLookup, error) {
	// CloudWatch Logs: ingestion + storage
	return nil, nil
}

func (h *LogGroupHandler) Describe(_ *pricing.Price, _ map[string]any) map[string]string { return nil }

func (h *LogGroupHandler) CalculateCost(_ *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
	// CloudWatch Logs: $0.50 per GB ingested, $0.03 per GB stored
	// Usage-based, no fixed cost
	return 0, 0
}

// AlarmHandler handles aws_cloudwatch_metric_alarm cost estimation
type AlarmHandler struct{}

func (h *AlarmHandler) Category() aws.CostCategory { return aws.CostCategoryFixed }

func (h *AlarmHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceCloudWatch
}

func (h *AlarmHandler) BuildLookup(_ string, _ map[string]any) (*pricing.PriceLookup, error) {
	return nil, nil
}

func (h *AlarmHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	desc := make(map[string]string)
	period := aws.GetIntAttr(attrs, "period")
	if period > 0 && period < HighResolutionThresholdSeconds {
		desc["resolution"] = "high"
	} else {
		desc["resolution"] = "standard"
	}
	return desc
}

func (h *AlarmHandler) CalculateCost(_ *pricing.Price, _ *pricing.PriceIndex, _ string, attrs map[string]any) (hourly, monthly float64) {
	// Standard resolution alarm: $0.10/alarm/month
	// High resolution alarm: $0.30/alarm/month
	period := aws.GetIntAttr(attrs, "period")
	if period > 0 && period < HighResolutionThresholdSeconds {
		return aws.FixedMonthlyCost(CloudWatchHighResAlarmCost)
	}
	return aws.FixedMonthlyCost(CloudWatchStandardAlarmCost)
}
