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

func TestScheduleReturnsValueGroups(t *testing.T) {
	t.Parallel()

	ir := &IR{jobs: []Job{{name: "plan-0", env: map[string]string{"A": "B"}}}}
	groups, err := Schedule(ir)
	if err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("groups = %d, want 1", len(groups))
	}

	jobs := groups[0].Jobs()
	if len(jobs) != 1 {
		t.Fatalf("jobs = %d, want 1", len(jobs))
	}
	jobs[0].name = "mutated"
	jobs[0].env["A"] = "changed"

	fresh := groups[0].Jobs()
	if got := fresh[0].Name(); got != "plan-0" {
		t.Fatalf("group job mutation leaked: got %q", got)
	}
	if got := fresh[0].Env()["A"]; got != "B" {
		t.Fatalf("group job env mutation leaked: got %q", got)
	}
	if got := ir.jobs[0].name; got != "plan-0" {
		t.Fatalf("IR job mutation leaked: got %q", got)
	}
}

func groupNames(groups []JobGroup) []string {
	names := make([]string, 0, len(groups))
	for _, group := range groups {
		names = append(names, group.Name())
	}
	return names
}

func jobNamesInGroup(group JobGroup) []string {
	jobs := group.Jobs()
	names := make([]string, 0, len(jobs))
	for i := range jobs {
		names = append(names, jobs[i].Name())
	}
	return names
}
