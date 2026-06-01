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
