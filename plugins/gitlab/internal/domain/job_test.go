package domain

import "testing"

func TestNewJobValidatesRequiredFields(t *testing.T) {
	if _, err := NewJob(JobOptions{Script: []string{"echo test"}}); err == nil {
		t.Fatal("NewJob() error = nil, want missing stage error")
	}
	if _, err := NewJob(JobOptions{Stage: "deploy"}); err == nil {
		t.Fatal("NewJob() error = nil, want missing script error")
	}
}
