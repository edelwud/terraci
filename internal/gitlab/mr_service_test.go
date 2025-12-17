package gitlab

import (
	"testing"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/pkg/config"
)

func TestExpandLabelPlaceholders(t *testing.T) {
	module := &discovery.Module{
		Service:     "platform",
		Environment: "stage",
		Region:      "eu-central-1",
		Module:      "vpc",
		Submodule:   "",
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "simple service",
			template: "{service}",
			expected: "platform",
		},
		{
			name:     "service and env",
			template: "{service}::{environment}",
			expected: "platform::stage",
		},
		{
			name:     "terraform label",
			template: "terraform",
			expected: "terraform",
		},
		{
			name:     "env shorthand",
			template: "{env}",
			expected: "stage",
		},
		{
			name:     "full module path",
			template: "{service}/{env}/{region}/{module}",
			expected: "platform/stage/eu-central-1/vpc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandLabelPlaceholders(tt.template, module)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExpandLabelPlaceholders_WithSubmodule(t *testing.T) {
	module := &discovery.Module{
		Service:     "platform",
		Environment: "stage",
		Region:      "eu-central-1",
		Module:      "ec2",
		Submodule:   "rabbitmq",
	}

	result := expandLabelPlaceholders("{module}/{submodule}", module)
	if result != "ec2/rabbitmq" {
		t.Errorf("expected 'ec2/rabbitmq', got %q", result)
	}
}

func TestMRService_IsEnabled(t *testing.T) {
	t.Run("not in MR", func(t *testing.T) {
		svc := &MRService{
			context: &MRContext{InMR: false},
			client:  &Client{token: "token"},
		}
		if svc.IsEnabled() {
			t.Error("expected IsEnabled to be false when not in MR")
		}
	})

	t.Run("in MR without token", func(t *testing.T) {
		svc := &MRService{
			context: &MRContext{InMR: true},
			client:  &Client{token: ""},
		}
		if svc.IsEnabled() {
			t.Error("expected IsEnabled to be false without token")
		}
	})

	t.Run("in MR with token, default config", func(t *testing.T) {
		svc := &MRService{
			context: &MRContext{InMR: true},
			client:  &Client{token: "token"},
			config:  nil,
		}
		if !svc.IsEnabled() {
			t.Error("expected IsEnabled to be true by default")
		}
	})

	t.Run("explicitly disabled", func(t *testing.T) {
		enabled := false
		svc := &MRService{
			context: &MRContext{InMR: true},
			client:  &Client{token: "token"},
			config: &config.MRConfig{
				Comment: &config.MRCommentConfig{
					Enabled: &enabled,
				},
			},
		}
		if svc.IsEnabled() {
			t.Error("expected IsEnabled to be false when explicitly disabled")
		}
	})
}

func TestModulesToPlans(t *testing.T) {
	modules := []*discovery.Module{
		{
			Service:      "platform",
			Environment:  "stage",
			Region:       "eu-central-1",
			Module:       "vpc",
			RelativePath: "platform/stage/eu-central-1/vpc",
		},
		{
			Service:      "platform",
			Environment:  "prod",
			Region:       "eu-central-1",
			Module:       "eks",
			RelativePath: "platform/prod/eu-central-1/eks",
		},
	}

	plans := ModulesToPlans(modules)

	if len(plans) != 2 {
		t.Fatalf("expected 2 plans, got %d", len(plans))
	}

	if plans[0].ModuleID != "platform/stage/eu-central-1/vpc" {
		t.Errorf("unexpected module ID: %s", plans[0].ModuleID)
	}

	if plans[0].Status != PlanStatusPending {
		t.Errorf("expected pending status, got %s", plans[0].Status)
	}

	if plans[1].Environment != "prod" {
		t.Errorf("expected prod environment, got %s", plans[1].Environment)
	}
}
