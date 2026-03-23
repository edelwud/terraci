package storage

import (
	"testing"

	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestS3Handler_Category(t *testing.T) {
	h := &S3Handler{}
	if h.Category() != aws.CostCategoryUsageBased {
		t.Errorf("Category() = %v, want CostCategoryUsageBased", h.Category())
	}
}

func TestS3Handler_ServiceCode(t *testing.T) {
	h := &S3Handler{}
	if h.ServiceCode() != pricing.ServiceS3 {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceS3)
	}
}

func TestS3Handler_BuildLookup(t *testing.T) {
	h := &S3Handler{}
	lookup, err := h.BuildLookup("us-east-1", nil)
	if err != nil {
		t.Fatalf("BuildLookup returned error: %v", err)
	}
	if lookup != nil {
		t.Errorf("BuildLookup = %v, want nil", lookup)
	}
}

func TestS3Handler_CalculateCost(t *testing.T) {
	h := &S3Handler{}
	hourly, monthly := h.CalculateCost(nil, nil, "", nil)
	if hourly != 0 {
		t.Errorf("hourly = %v, want 0", hourly)
	}
	if monthly != 0 {
		t.Errorf("monthly = %v, want 0", monthly)
	}
}

func TestS3Handler_Describe(t *testing.T) {
	h := &S3Handler{}
	result := h.Describe(nil, nil)
	if result != nil {
		t.Errorf("Describe() = %v, want nil", result)
	}

	result = h.Describe(nil, map[string]any{"bucket": "my-bucket"})
	if result != nil {
		t.Errorf("Describe() with attrs = %v, want nil", result)
	}
}
