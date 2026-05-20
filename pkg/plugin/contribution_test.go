package plugin

import (
	"errors"
	"testing"
)

func TestPipelineContributionError(t *testing.T) {
	inner := errors.New("build failed")
	err := &PipelineContributionError{
		Plugin: "cost",
		Phase:  PipelineContributionPhaseContribution,
		Err:    inner,
	}

	if got, want := err.Error(), `pipeline contribution from plugin "cost" during contribution: build failed`; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	if !errors.Is(err, inner) {
		t.Fatalf("errors.Is() did not match wrapped error")
	}

	var typed *PipelineContributionError
	if !errors.As(err, &typed) {
		t.Fatalf("errors.As() did not match PipelineContributionError")
	}
	if typed.Plugin != "cost" || typed.Phase != PipelineContributionPhaseContribution {
		t.Fatalf("typed error = %#v", typed)
	}
}
