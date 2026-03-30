package handler

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// stubHandler implements ResourceHandler for testing the registry.
type stubHandler struct {
	svc      pricing.ServiceID
	category CostCategory
}

func (h *stubHandler) Category() CostCategory { return h.category }
func (h *stubHandler) BuildLookup(string, map[string]any) (*pricing.PriceLookup, error) {
	return &pricing.PriceLookup{ServiceID: h.svc}, nil
}
func (h *stubHandler) CalculateCost(*pricing.Price, *pricing.PriceIndex, string, map[string]any) (hourly, monthly float64) {
	return 0, 0
}
func (h *stubHandler) Describe(*pricing.Price, map[string]any) map[string]string { return nil }

func newTestRegistry() *Registry {
	r := NewRegistry()
	r.Register(awskit.ProviderID, ResourceType("aws_instance"), &stubHandler{svc: awskit.MustService(awskit.ServiceKeyEC2), category: CostCategoryStandard})
	r.Register(awskit.ProviderID, ResourceType("aws_ebs_volume"), &stubHandler{svc: awskit.MustService(awskit.ServiceKeyEC2), category: CostCategoryStandard})
	r.Register(awskit.ProviderID, ResourceType("aws_db_instance"), &stubHandler{svc: awskit.MustService(awskit.ServiceKeyRDS), category: CostCategoryStandard})
	r.Register(awskit.ProviderID, ResourceType("aws_lb"), &stubHandler{svc: awskit.MustService(awskit.ServiceKeyEC2), category: CostCategoryStandard})
	r.Register(awskit.ProviderID, ResourceType("aws_alb"), &stubHandler{svc: awskit.MustService(awskit.ServiceKeyEC2), category: CostCategoryStandard})
	r.Register(awskit.ProviderID, ResourceType("aws_elasticache_cluster"), &stubHandler{svc: awskit.MustService(awskit.ServiceKeyElastiCache), category: CostCategoryStandard})
	r.Register(awskit.ProviderID, ResourceType("aws_eks_cluster"), &stubHandler{svc: awskit.MustService(awskit.ServiceKeyEKS), category: CostCategoryStandard})
	r.Register(awskit.ProviderID, ResourceType("aws_lambda_function"), &stubHandler{svc: awskit.MustService(awskit.ServiceKeyLambda), category: CostCategoryStandard})
	r.Register(awskit.ProviderID, ResourceType("aws_dynamodb_table"), &stubHandler{svc: awskit.MustService(awskit.ServiceKeyDynamoDB), category: CostCategoryStandard})
	return r
}

func TestRegistry_ResolveHandler(t *testing.T) {
	t.Parallel()

	r := newTestRegistry()

	h, ok := r.ResolveHandler(awskit.ProviderID, ResourceType("aws_instance"))
	if !ok {
		t.Fatal("ResolveHandler should return handler for aws_instance")
	}
	if h == nil {
		t.Fatal("Handler should not be nil")
	}
	lookupBuilder, ok := h.(LookupBuilder)
	if !ok {
		t.Fatal("aws_instance handler should implement LookupBuilder")
	}
	lookup, err := lookupBuilder.BuildLookup("us-east-1", nil)
	if err != nil {
		t.Fatalf("BuildLookup() error = %v", err)
	}
	if lookup.ServiceID != awskit.MustService(awskit.ServiceKeyEC2) {
		t.Errorf("aws_instance lookup service = %q, want %q", lookup.ServiceID, awskit.MustService(awskit.ServiceKeyEC2))
	}

	_, ok = r.ResolveHandler(awskit.ProviderID, ResourceType("aws_nonexistent_resource"))
	if ok {
		t.Error("ResolveHandler should return false for nonexistent resource")
	}
}

func TestRegistry_ResolveHandler_ForRegisteredType(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	r.Register(awskit.ProviderID, ResourceType("aws_instance"), &stubHandler{svc: awskit.MustService(awskit.ServiceKeyEC2), category: CostCategoryStandard})

	h, ok := r.ResolveHandler(awskit.ProviderID, ResourceType("aws_instance"))
	if !ok || h == nil {
		t.Fatal("ResolveHandler should return registered handler")
	}
}

func TestRegistry_IsSupported(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			if r.IsSupported(ResourceType(tt.resourceType)) != tt.expected {
				t.Errorf("IsSupported(%q) = %v, want %v", tt.resourceType, !tt.expected, tt.expected)
			}
		})
	}
}

func TestRegistry_SupportedTypes(t *testing.T) {
	t.Parallel()

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

func TestNewRegistry(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry should return non-nil registry")
	}
	if len(r.SupportedTypes()) != 0 {
		t.Error("SupportedTypes should be empty for new registry")
	}

	r.Register(awskit.ProviderID, ResourceType("aws_test_resource"), &stubHandler{svc: awskit.MustService(awskit.ServiceKeyEC2), category: CostCategoryStandard})
	if !r.IsSupported(ResourceType("aws_test_resource")) {
		t.Error("should support registered resource")
	}
}

func TestLogUnsupported(_ *testing.T) {
	// Just verify it doesn't panic
	LogUnsupported("aws_unknown_resource", "module.foo.aws_unknown_resource.bar")
	LogUnsupported("", "")
}
