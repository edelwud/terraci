package summaryengine

import (
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
)

func TestHasReportableChanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		plans  []ci.ModulePlan
		policy *ci.PolicySummary
		want   bool
	}{
		{
			name:   "plans with changes status",
			plans:  []ci.ModulePlan{{Status: ci.PlanStatusChanges}},
			policy: nil,
			want:   true,
		},
		{
			name:   "plans with failed status",
			plans:  []ci.ModulePlan{{Status: ci.PlanStatusFailed}},
			policy: nil,
			want:   true,
		},
		{
			name:   "plans with no_changes only",
			plans:  []ci.ModulePlan{{Status: ci.PlanStatusNoChanges}},
			policy: nil,
			want:   false,
		},
		{
			name:   "empty plans nil policy",
			plans:  []ci.ModulePlan{},
			policy: nil,
			want:   false,
		},
		{
			name:   "nil plans nil policy",
			plans:  nil,
			policy: nil,
			want:   false,
		},
		{
			name:  "nil plans but policy with failures",
			plans: nil,
			policy: &ci.PolicySummary{
				FailedModules: 1,
				TotalFailures: 2,
			},
			want: true,
		},
		{
			name:  "nil plans but policy with warnings",
			plans: nil,
			policy: &ci.PolicySummary{
				WarnedModules: 1,
				TotalWarnings: 3,
			},
			want: true,
		},
		{
			name:  "nil plans policy with no failures or warnings",
			plans: nil,
			policy: &ci.PolicySummary{
				TotalModules:  2,
				PassedModules: 2,
			},
			want: false,
		},
		{
			name: "mixed statuses with one change",
			plans: []ci.ModulePlan{
				{Status: ci.PlanStatusNoChanges},
				{Status: ci.PlanStatusSuccess},
				{Status: ci.PlanStatusChanges},
			},
			policy: nil,
			want:   true,
		},
		{
			name: "all non-reportable statuses",
			plans: []ci.ModulePlan{
				{Status: ci.PlanStatusNoChanges},
				{Status: ci.PlanStatusSuccess},
				{Status: ci.PlanStatusPending},
			},
			policy: nil,
			want:   false,
		},
		{
			name:  "no plan changes but policy has TotalWarnings only",
			plans: []ci.ModulePlan{{Status: ci.PlanStatusNoChanges}},
			policy: &ci.PolicySummary{
				TotalWarnings: 1,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := HasReportableChanges(tt.plans, tt.policy)
			if got != tt.want {
				t.Errorf("HasReportableChanges() = %v, want %v", got, tt.want)
			}
		})
	}
}
