package model_test

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

func TestDetectRegion(t *testing.T) {
	t.Parallel()

	segments := []string{"service", "environment", "region", "module"}

	tests := []struct {
		name       string
		segments   []string
		modulePath string
		want       string
	}{
		{"extracts region from pattern", segments, "platform/prod/eu-central-1/rds", "eu-central-1"},
		{"extracts us-east-1", segments, "svc/staging/us-east-1/vpc", "us-east-1"},
		{"falls back when no region segment", []string{"service", "module"}, "svc/vpc", "us-east-1"},
		{"falls back on nil segments", nil, "svc/prod/eu-west-1/vpc", "us-east-1"},
		{"falls back when path too short", segments, "svc", "us-east-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := model.DetectRegion(tt.segments, tt.modulePath)
			if got != tt.want {
				t.Errorf("DetectRegion(%v, %q) = %q, want %q", tt.segments, tt.modulePath, got, tt.want)
			}
		})
	}
}
