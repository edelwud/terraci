package pipeline

import "testing"

func TestScheduleUsesTopologicalLayers(t *testing.T) {
	t.Parallel()

	ir := &IR{
		Jobs: []Job{
			{Name: "plan-0"},
			{Name: "apply-0", Dependencies: []JobDependency{{Job: "plan-0"}}},
			{Name: "plan-1", Dependencies: []JobDependency{{Job: "apply-0"}}},
			{Name: "apply-1", Dependencies: []JobDependency{{Job: "plan-1"}}},
			{Name: "tfupdate"},
			{Name: "policy", Dependencies: []JobDependency{{Job: "plan-1"}}},
			{Name: "summary", Dependencies: []JobDependency{{Job: "policy"}, {Job: "apply-1"}}},
		},
	}

	groups, err := Schedule(ir)
	if err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}

	want := [][]string{
		{"plan-0", "tfupdate"},
		{"apply-0"},
		{"plan-1"},
		{"apply-1", "policy"},
		{"summary"},
	}
	if len(groups) != len(want) {
		t.Fatalf("groups = %v, want %v", groupNames(groups), want)
	}
	for i := range want {
		got := jobNamesInGroup(groups[i])
		if len(got) != len(want[i]) {
			t.Fatalf("group %d jobs = %v, want %v", i, got, want[i])
		}
		for j := range want[i] {
			if got[j] != want[i][j] {
				t.Fatalf("group %d jobs = %v, want %v", i, got, want[i])
			}
		}
	}
}

func TestScheduleRejectsUnknownDependency(t *testing.T) {
	t.Parallel()

	_, err := Schedule(&IR{Jobs: []Job{{Name: "summary", Dependencies: []JobDependency{{Job: "missing"}}}}})
	if err == nil {
		t.Fatal("Schedule() error = nil, want unknown dependency error")
	}
}

func TestScheduleRejectsCycle(t *testing.T) {
	t.Parallel()

	_, err := Schedule(&IR{Jobs: []Job{
		{Name: "a", Dependencies: []JobDependency{{Job: "b"}}},
		{Name: "b", Dependencies: []JobDependency{{Job: "a"}}},
	}})
	if err == nil {
		t.Fatal("Schedule() error = nil, want cycle error")
	}
}

func groupNames(groups []JobGroup) []string {
	names := make([]string, 0, len(groups))
	for _, group := range groups {
		names = append(names, group.Name)
	}
	return names
}

func jobNamesInGroup(group JobGroup) []string {
	names := make([]string, 0, len(group.Jobs))
	for _, job := range group.Jobs {
		names = append(names, job.Name)
	}
	return names
}
