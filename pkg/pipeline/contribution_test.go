package pipeline

import (
	"strings"
	"testing"
)

func TestNewContributedJobValidatesAndCopiesInput(t *testing.T) {
	t.Parallel()

	commands := []string{"terraci cost"}
	consumes := []ResourceRequest{AllPlanResources(ResourceKindPlanJSON)}
	produces := PluginResultAndReportResources(".terraci", "cost")

	job, err := NewContributedJob(ContributedJobOptions{
		Name:         "cost-estimation",
		Commands:     commands,
		Consumes:     consumes,
		Produces:     produces,
		AllowFailure: true,
	})
	if err != nil {
		t.Fatalf("NewContributedJob() error = %v", err)
	}

	commands[0] = "changed"
	consumes[0] = ResourceRequest{}
	produces[0].Path = "changed"
	if got := job.Commands()[0]; got != "terraci cost" {
		t.Fatalf("Commands() = %q, want original", got)
	}
	if got := job.Consumes()[0].Kind(); got != ResourceKindPlanJSON {
		t.Fatalf("Consumes()[0].Kind = %q, want plan_json", got)
	}
	if got := job.Produces()[0].Path; got == "changed" {
		t.Fatal("Produces() leaked input mutation")
	}

	returnedCommands := job.Commands()
	returnedCommands[0] = "mutated"
	if got := job.Commands()[0]; got != "terraci cost" {
		t.Fatalf("Commands() leaked returned slice mutation: %q", got)
	}
}

func TestNewContributedJobRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    ContributedJobOptions
		wantErr string
	}{
		{
			name:    "name required",
			opts:    ContributedJobOptions{Commands: []string{"check"}},
			wantErr: "name is required",
		},
		{
			name:    "commands required",
			opts:    ContributedJobOptions{Name: "check"},
			wantErr: "requires at least one command",
		},
		{
			name:    "empty command",
			opts:    ContributedJobOptions{Name: "check", Commands: []string{" "}},
			wantErr: "commands[0] is empty",
		},
		{
			name: "invalid consume",
			opts: ContributedJobOptions{
				Name:     "check",
				Commands: []string{"check"},
				Consumes: []ResourceRequest{{kind: ResourceKindPluginReport}},
			},
			wantErr: "selector scope is required",
		},
		{
			name: "invalid produce path",
			opts: ContributedJobOptions{
				Name:     "check",
				Commands: []string{"check"},
				Produces: []ResourceSpec{
					PluginResource(ResourceKindPluginReport, "policy", "../policy-report.json"),
				},
			},
			wantErr: "path invalid",
		},
		{
			name: "duplicate output",
			opts: ContributedJobOptions{
				Name:     "check",
				Commands: []string{"check"},
				Produces: []ResourceSpec{
					PluginResource(ResourceKindPluginReport, "policy", ".terraci/policy-report.json"),
					PluginResource(ResourceKindPluginReport, "policy", ".terraci/other-report.json"),
				},
			},
			wantErr: "duplicate resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewContributedJob(tt.opts)
			if err == nil {
				t.Fatal("NewContributedJob() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("NewContributedJob() error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestNewContributionValidatesDuplicatesAndReturnsDefensiveCopies(t *testing.T) {
	t.Parallel()

	job, err := NewContributedJob(ContributedJobOptions{
		Name:     "summary",
		Commands: []string{"terraci summary"},
	})
	if err != nil {
		t.Fatalf("NewContributedJob() error = %v", err)
	}
	contribution, err := NewContribution(job)
	if err != nil {
		t.Fatalf("NewContribution() error = %v", err)
	}

	jobs := contribution.Jobs()
	jobs[0] = ContributedJob{}
	if got := contribution.Jobs()[0].Name(); got != "summary" {
		t.Fatalf("Jobs() leaked mutation: %q", got)
	}

	clone := contribution.Clone()
	cloneJobs := clone.Jobs()
	cloneJobs[0] = ContributedJob{}
	if got := contribution.Jobs()[0].Name(); got != "summary" {
		t.Fatalf("Clone() leaked mutation: %q", got)
	}
}

func TestNewContributionRejectsDuplicateOutputs(t *testing.T) {
	t.Parallel()

	first, err := NewContributedJob(ContributedJobOptions{
		Name:     "policy-a",
		Commands: []string{"policy-a"},
		Produces: []ResourceSpec{
			PluginResource(ResourceKindPluginReport, "policy", ".terraci/policy-report.json"),
		},
	})
	if err != nil {
		t.Fatalf("NewContributedJob(first) error = %v", err)
	}
	second, err := NewContributedJob(ContributedJobOptions{
		Name:     "policy-b",
		Commands: []string{"policy-b"},
		Produces: []ResourceSpec{
			PluginResource(ResourceKindPluginReport, "policy", ".terraci/policy-report.json"),
		},
	})
	if err != nil {
		t.Fatalf("NewContributedJob(second) error = %v", err)
	}

	_, err = NewContribution(first, second)
	if err == nil {
		t.Fatal("NewContribution() error = nil, want duplicate output error")
	}
	if !strings.Contains(err.Error(), "duplicate produced resource") {
		t.Fatalf("NewContribution() error = %q, want duplicate output", err.Error())
	}
}

func TestContributionSetCopiesAndFlattensJobs(t *testing.T) {
	t.Parallel()

	job, err := NewContributedJob(ContributedJobOptions{
		Name:     "summary",
		Commands: []string{"terraci summary"},
	})
	if err != nil {
		t.Fatalf("NewContributedJob() error = %v", err)
	}
	contribution, err := NewContribution(job)
	if err != nil {
		t.Fatalf("NewContribution() error = %v", err)
	}
	set, err := NewContributionSet(contribution)
	if err != nil {
		t.Fatalf("NewContributionSet() error = %v", err)
	}

	if set.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", set.Len())
	}
	if set.IsEmpty() {
		t.Fatal("IsEmpty() = true, want false")
	}
	jobs := set.Jobs()
	if len(jobs) != 1 || jobs[0].Name() != "summary" {
		t.Fatalf("Jobs() = %#v, want summary job", jobs)
	}

	items := set.Contributions()
	if len(items) != 1 || items[0] == contribution {
		t.Fatalf("Contributions() = %#v, want defensive contribution copy", items)
	}
	items[0] = nil
	if again := set.Contributions(); len(again) != 1 || again[0] == nil {
		t.Fatalf("Contributions() leaked returned slice mutation: %#v", again)
	}

	clone := set.Clone()
	cloneItems := clone.Contributions()
	cloneItems[0] = nil
	if again := set.Contributions(); len(again) != 1 || again[0] == nil {
		t.Fatalf("Clone() leaked mutation: %#v", again)
	}
}

func TestContributionSetEmptyAndNilValidation(t *testing.T) {
	t.Parallel()

	if !EmptyContributionSet().IsEmpty() {
		t.Fatal("EmptyContributionSet().IsEmpty() = false, want true")
	}
	var zero ContributionSet
	if zero.Len() != 0 || !zero.IsEmpty() {
		t.Fatalf("zero ContributionSet len/empty = %d/%t, want 0/true", zero.Len(), zero.IsEmpty())
	}
	if jobs := zero.Jobs(); jobs != nil {
		t.Fatalf("zero ContributionSet Jobs() = %#v, want nil", jobs)
	}
	if _, err := NewContributionSet(nil); err == nil {
		t.Fatal("NewContributionSet(nil) error = nil, want error")
	}
}
