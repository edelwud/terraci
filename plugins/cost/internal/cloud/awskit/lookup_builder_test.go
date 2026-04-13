package awskit

import "testing"

func TestLookupBuilder_Build(t *testing.T) {
	t.Parallel()

	runtime := NewRuntime(Manifest)
	lookup := runtime.
		NewLookupBuilder(ServiceKeyEC2, "Compute Instance").
		Attr("instanceType", "m5.large").
		AttrIf(false, "tenancy", "Dedicated").
		UsageType("us-east-1", "BoxUsage:m5.large").
		Build("us-east-1")

	if lookup.ServiceID != runtime.MustService(ServiceKeyEC2) {
		t.Fatalf("service = %#v, want %#v", lookup.ServiceID, runtime.MustService(ServiceKeyEC2))
	}
	if lookup.ProductFamily != "Compute Instance" {
		t.Fatalf("product family = %q, want %q", lookup.ProductFamily, "Compute Instance")
	}
	if lookup.Attributes["instanceType"] != "m5.large" {
		t.Fatalf("instanceType = %q, want %q", lookup.Attributes["instanceType"], "m5.large")
	}
	if _, ok := lookup.Attributes["tenancy"]; ok {
		t.Fatalf("tenancy should not be present: %#v", lookup.Attributes)
	}
	if lookup.Attributes["usagetype"] == "" {
		t.Fatalf("usagetype should be populated: %#v", lookup.Attributes)
	}
	if lookup.Attributes["location"] == "" {
		t.Fatalf("location should be populated: %#v", lookup.Attributes)
	}
}

func TestMatchString(t *testing.T) {
	t.Parallel()

	got := MatchString("network", "Load Balancer", map[string]string{
		"network": "Load Balancer-Network",
		"gateway": "Load Balancer-Gateway",
	})
	if got != "Load Balancer-Network" {
		t.Fatalf("MatchString() = %q, want %q", got, "Load Balancer-Network")
	}
}

func TestLookupBuilder_ProductFamilyMatch(t *testing.T) {
	t.Parallel()

	runtime := NewRuntime(Manifest)
	lookup := runtime.
		NewLookupBuilder(ServiceKeyEC2, "").
		ProductFamilyMatch("application", "Load Balancer", map[string]string{
			"network": "Load Balancer-Network",
		}).
		Build("us-east-1")

	if lookup.ProductFamily != "Load Balancer" {
		t.Fatalf("product family = %q, want %q", lookup.ProductFamily, "Load Balancer")
	}
}

func TestLookupBuilder_AttrMatch(t *testing.T) {
	t.Parallel()

	runtime := NewRuntime(Manifest)
	lookup := runtime.
		NewLookupBuilder(ServiceKeyEC2, "Compute Instance").
		AttrMatch("tenancy", "host", "Shared", map[string]string{
			"":          "Shared",
			"default":   "Shared",
			"dedicated": "Dedicated",
			"host":      "Host",
		}).
		Build("us-east-1")

	if lookup.Attributes["tenancy"] != "Host" {
		t.Fatalf("tenancy = %q, want %q", lookup.Attributes["tenancy"], "Host")
	}
}
