package aws

import (
	"testing"

	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()

	// Check some expected handlers are registered
	expectedTypes := []string{
		"aws_instance",
		"aws_ebs_volume",
		"aws_db_instance",
		"aws_lb",
		"aws_elasticache_cluster",
		"aws_eks_cluster",
		"aws_lambda_function",
		"aws_dynamodb_table",
	}

	for _, rt := range expectedTypes {
		if !r.IsSupported(rt) {
			t.Errorf("Registry should support %q", rt)
		}
	}
}

func TestRegistry_GetHandler(t *testing.T) {
	r := NewRegistry()

	// Test existing handler
	handler, ok := r.GetHandler("aws_instance")
	if !ok {
		t.Fatal("GetHandler should return handler for aws_instance")
	}
	if handler == nil {
		t.Fatal("Handler should not be nil")
	}
	if handler.ServiceCode() != pricing.ServiceEC2 {
		t.Errorf("aws_instance ServiceCode = %q, want %q", handler.ServiceCode(), pricing.ServiceEC2)
	}

	// Test non-existing handler
	_, ok = r.GetHandler("aws_nonexistent_resource")
	if ok {
		t.Error("GetHandler should return false for nonexistent resource")
	}
}

func TestRegistry_IsSupported(t *testing.T) {
	r := NewRegistry()

	tests := []struct {
		resourceType string
		expected     bool
	}{
		{"aws_instance", true},
		{"aws_db_instance", true},
		{"aws_lb", true},
		{"aws_alb", true}, // alias
		{"aws_nonexistent", false},
		{"google_compute_instance", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			result := r.IsSupported(tt.resourceType)
			if result != tt.expected {
				t.Errorf("IsSupported(%q) = %v, want %v", tt.resourceType, result, tt.expected)
			}
		})
	}
}

func TestRegistry_SupportedTypes(t *testing.T) {
	r := NewRegistry()
	types := r.SupportedTypes()

	if len(types) == 0 {
		t.Error("SupportedTypes should return non-empty list")
	}

	// Check at least some expected types are present
	typeSet := make(map[string]bool)
	for _, t := range types {
		typeSet[t] = true
	}

	if !typeSet["aws_instance"] {
		t.Error("SupportedTypes should include aws_instance")
	}
	if !typeSet["aws_db_instance"] {
		t.Error("SupportedTypes should include aws_db_instance")
	}
}

func TestRegistry_RequiredServices(t *testing.T) {
	r := NewRegistry()

	resourceTypes := []string{"aws_instance", "aws_db_instance", "aws_elasticache_cluster"}
	services := r.RequiredServices(resourceTypes)

	if len(services) == 0 {
		t.Error("RequiredServices should return non-empty map")
	}

	if !services[pricing.ServiceEC2] {
		t.Error("RequiredServices should include ServiceEC2 for aws_instance")
	}
	if !services[pricing.ServiceRDS] {
		t.Error("RequiredServices should include ServiceRDS for aws_db_instance")
	}
	if !services[pricing.ServiceElastiCache] {
		t.Error("RequiredServices should include ServiceElastiCache for aws_elasticache_cluster")
	}
}
