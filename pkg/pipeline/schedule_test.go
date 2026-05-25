package pipeline

import "testing"

func TestScheduleUsesTopologicalLayers(t *testing.T) {
	t.Parallel()

	ir := &IR{
		jobs: []Job{
			{name: "plan-0"},
			{name: "apply-0", dependencies: []JobDependency{{Job: "plan-0"}}},
			{name: "plan-1", dependencies: []JobDependency{{Job: "apply-0"}}},
			{name: "apply-1", dependencies: []JobDependency{{Job: "plan-1"}}},
			{name: "tfupdate"},
			{name: "policy", dependencies: []JobDependency{{Job: "plan-1"}}},
			{name: "summary", dependencies: []JobDependency{{Job: "policy"}, {Job: "apply-1"}}},
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

	_, err := Schedule(&IR{jobs: []Job{{name: "summary", dependencies: []JobDependency{{Job: "missing"}}}}})
	if err == nil {
		t.Fatal("Schedule() error = nil, want unknown dependency error")
	}
}

func TestScheduleRejectsCycle(t *testing.T) {
	t.Parallel()

	_, err := Schedule(&IR{jobs: []Job{
		{name: "a", dependencies: []JobDependency{{Job: "b"}}},
		{name: "b", dependencies: []JobDependency{{Job: "a"}}},
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
		names = append(names, job.name)
	}
	return names
}
