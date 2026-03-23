package ci

import "testing"

func TestHasReportableChanges(t *testing.T) {
	tests := []struct {
		name   string
		plans  []ModulePlan
		policy *PolicySummary
		want   bool
	}{
		{
			name:   "plans with changes status",
			plans:  []ModulePlan{{Status: PlanStatusChanges}},
			policy: nil,
			want:   true,
		},
		{
			name:   "plans with failed status",
			plans:  []ModulePlan{{Status: PlanStatusFailed}},
			policy: nil,
			want:   true,
		},
		{
			name:   "plans with no_changes only",
			plans:  []ModulePlan{{Status: PlanStatusNoChanges}},
			policy: nil,
			want:   false,
		},
		{
			name:   "empty plans nil policy",
			plans:  []ModulePlan{},
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
			policy: &PolicySummary{
				FailedModules: 1,
				TotalFailures: 2,
			},
			want: true,
		},
		{
			name:  "nil plans but policy with warnings",
			plans: nil,
			policy: &PolicySummary{
				WarnedModules: 1,
				TotalWarnings: 3,
			},
			want: true,
		},
		{
			name:  "nil plans policy with no failures or warnings",
			plans: nil,
			policy: &PolicySummary{
				TotalModules:  2,
				PassedModules: 2,
			},
			want: false,
		},
		{
			name: "mixed statuses with one change",
			plans: []ModulePlan{
				{Status: PlanStatusNoChanges},
				{Status: PlanStatusSuccess},
				{Status: PlanStatusChanges},
			},
			policy: nil,
			want:   true,
		},
		{
			name: "all non-reportable statuses",
			plans: []ModulePlan{
				{Status: PlanStatusNoChanges},
				{Status: PlanStatusSuccess},
				{Status: PlanStatusPending},
			},
			policy: nil,
			want:   false,
		},
		{
			name:  "no plan changes but policy has TotalWarnings only",
			plans: []ModulePlan{{Status: PlanStatusNoChanges}},
			policy: &PolicySummary{
				TotalWarnings: 1,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasReportableChanges(tt.plans, tt.policy)
			if got != tt.want {
				t.Errorf("HasReportableChanges() = %v, want %v", got, tt.want)
			}
		})
	}
}
