package aws

import (
	"testing"

	"github.com/edelwud/terraci/internal/cost/pricing"
)

// stubHandler implements ResourceHandler for testing the registry.
type stubHandler struct {
	svc      pricing.ServiceCode
	category CostCategory
}

func (h *stubHandler) Category() CostCategory           { return h.category }
func (h *stubHandler) ServiceCode() pricing.ServiceCode { return h.svc }
func (h *stubHandler) BuildLookup(string, map[string]any) (*pricing.PriceLookup, error) {
	return nil, nil
}
func (h *stubHandler) CalculateCost(*pricing.Price, *pricing.PriceIndex, string, map[string]any) (hourly, monthly float64) {
	return 0, 0
}
func (h *stubHandler) Describe(*pricing.Price, map[string]any) map[string]string { return nil }

func newTestRegistry() *Registry {
	r := &Registry{handlers: make(map[string]ResourceHandler)}
	r.Register("aws_instance", &stubHandler{svc: pricing.ServiceEC2, category: CostCategoryStandard})
	r.Register("aws_ebs_volume", &stubHandler{svc: pricing.ServiceEC2, category: CostCategoryStandard})
	r.Register("aws_db_instance", &stubHandler{svc: pricing.ServiceRDS, category: CostCategoryStandard})
	r.Register("aws_lb", &stubHandler{svc: pricing.ServiceEC2, category: CostCategoryStandard})
	r.Register("aws_alb", &stubHandler{svc: pricing.ServiceEC2, category: CostCategoryStandard})
	r.Register("aws_elasticache_cluster", &stubHandler{svc: pricing.ServiceElastiCache, category: CostCategoryStandard})
	r.Register("aws_eks_cluster", &stubHandler{svc: pricing.ServiceEKS, category: CostCategoryStandard})
	r.Register("aws_lambda_function", &stubHandler{svc: pricing.ServiceLambda, category: CostCategoryStandard})
	r.Register("aws_dynamodb_table", &stubHandler{svc: pricing.ServiceDynamoDB, category: CostCategoryStandard})
	return r
}

func TestRegistry_GetHandler(t *testing.T) {
	r := newTestRegistry()

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

	_, ok = r.GetHandler("aws_nonexistent_resource")
	if ok {
		t.Error("GetHandler should return false for nonexistent resource")
	}
}

func TestRegistry_IsSupported(t *testing.T) {
	r := newTestRegistry()

	tests := []struct {
		resourceType string
		expected     bool
	}{
		{"aws_instance", true},
		{"aws_db_instance", true},
		{"aws_lb", true},
		{"aws_alb", true},
		{"aws_nonexistent", false},
		{"google_compute_instance", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			if r.IsSupported(tt.resourceType) != tt.expected {
				t.Errorf("IsSupported(%q) = %v, want %v", tt.resourceType, !tt.expected, tt.expected)
			}
		})
	}
}

func TestRegistry_SupportedTypes(t *testing.T) {
	r := newTestRegistry()
	types := r.SupportedTypes()

	if len(types) == 0 {
		t.Error("SupportedTypes should return non-empty list")
	}

	typeSet := make(map[string]bool)
	for _, tp := range types {
		typeSet[tp] = true
	}

	if !typeSet["aws_instance"] {
		t.Error("SupportedTypes should include aws_instance")
	}
	if !typeSet["aws_db_instance"] {
		t.Error("SupportedTypes should include aws_db_instance")
	}
}

func TestRegistry_RequiredServices(t *testing.T) {
	r := newTestRegistry()

	services := r.RequiredServices([]string{"aws_instance", "aws_db_instance", "aws_elasticache_cluster"})

	if len(services) == 0 {
		t.Error("RequiredServices should return non-empty map")
	}
	if !services[pricing.ServiceEC2] {
		t.Error("should include ServiceEC2")
	}
	if !services[pricing.ServiceRDS] {
		t.Error("should include ServiceRDS")
	}
	if !services[pricing.ServiceElastiCache] {
		t.Error("should include ServiceElastiCache")
	}
}
