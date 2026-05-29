package domain

import "testing"

func TestNewJobValidatesRequiredFields(t *testing.T) {
	if _, err := NewJob(JobOptions{Steps: []Step{NewStep(StepOptions{Name: "Checkout", Uses: "actions/checkout@v4"})}}); err == nil {
		t.Fatal("NewJob() error = nil, want missing runs-on error")
	}
	if _, err := NewJob(JobOptions{RunsOn: "ubuntu-latest"}); err == nil {
		t.Fatal("NewJob() error = nil, want missing steps error")
	}
}
