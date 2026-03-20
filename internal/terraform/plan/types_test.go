package plan

import "testing"

func TestHasChanges(t *testing.T) {
	tests := []struct {
		name string
		plan ParsedPlan
		want bool
	}{
		{"no changes", ParsedPlan{}, false},
		{"has add", ParsedPlan{ToAdd: 1}, true},
		{"has change", ParsedPlan{ToChange: 1}, true},
		{"has destroy", ParsedPlan{ToDestroy: 1}, true},
		{"has all", ParsedPlan{ToAdd: 1, ToChange: 2, ToDestroy: 3}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.plan.HasChanges(); got != tt.want {
				t.Errorf("HasChanges() = %v, want %v", got, tt.want)
			}
		})
	}
}
