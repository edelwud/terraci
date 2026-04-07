package awskit

import "testing"

func TestPriceLookupSpec_Lookup(t *testing.T) {
	spec := PriceLookupSpec{
		Service:       MustService(ServiceKeyEC2),
		ProductFamily: "Compute Instance",
	}

	lookup := spec.Lookup("us-east-1", map[string]string{
		"instanceType": "t3.micro",
		"tenancy":      "Shared",
	})

	if lookup.ServiceID != MustService(ServiceKeyEC2) {
		t.Errorf("ServiceCode = %v, want %v", lookup.ServiceID, MustService(ServiceKeyEC2))
	}
	if lookup.Region != "us-east-1" {
		t.Errorf("Region = %q, want %q", lookup.Region, "us-east-1")
	}
	if lookup.ProductFamily != "Compute Instance" {
		t.Errorf("ProductFamily = %q, want %q", lookup.ProductFamily, "Compute Instance")
	}
	if lookup.Attributes["instanceType"] != "t3.micro" {
		t.Errorf("instanceType = %q, want %q", lookup.Attributes["instanceType"], "t3.micro")
	}
	if lookup.Attributes["tenancy"] != "Shared" {
		t.Errorf("tenancy = %q, want %q", lookup.Attributes["tenancy"], "Shared")
	}
}

func TestPriceLookupSpec_Lookup_AddsLocation(t *testing.T) {
	spec := PriceLookupSpec{
		Service:       MustService(ServiceKeyRDS),
		ProductFamily: "Database Instance",
	}

	lookup := spec.Lookup("eu-west-1", map[string]string{
		"instanceType": "db.t3.medium",
	})

	location := lookup.Attributes["location"]
	if location == "" {
		t.Fatal("location attribute missing")
	}
	if location == "eu-west-1" {
		t.Error("location should be the resolved name, not the raw region code")
	}
	// Should be the full AWS region name
	expected := DefaultRuntime.ResolveRegionName("eu-west-1")
	if location != expected {
		t.Errorf("location = %q, want %q", location, expected)
	}
}

func TestPriceLookupSpec_Lookup_NilAttrs(t *testing.T) {
	spec := PriceLookupSpec{
		Service:       MustService(ServiceKeyEC2),
		ProductFamily: "NAT Gateway",
	}

	lookup := spec.Lookup("us-west-2", nil)

	if lookup.Attributes == nil {
		t.Fatal("Attributes should not be nil")
	}
	if lookup.Attributes["location"] == "" {
		t.Error("location should be set even with nil attrs")
	}
}

func TestPriceLookupSpec_Lookup_UnknownRegion(t *testing.T) {
	spec := PriceLookupSpec{
		Service:       MustService(ServiceKeyEC2),
		ProductFamily: "Storage",
	}

	lookup := spec.Lookup("xx-unknown-1", nil)

	// Unknown region falls back to raw code
	if lookup.Attributes["location"] != "xx-unknown-1" {
		t.Errorf("location = %q, want %q for unknown region", lookup.Attributes["location"], "xx-unknown-1")
	}
	if lookup.Region != "xx-unknown-1" {
		t.Errorf("Region = %q, want %q", lookup.Region, "xx-unknown-1")
	}
}

func TestPriceLookupSpec_Lookup_DoesNotOverwriteExistingAttrs(t *testing.T) {
	spec := PriceLookupSpec{
		Service:       MustService(ServiceKeyEC2),
		ProductFamily: "Compute Instance",
	}

	attrs := map[string]string{
		"instanceType": "m5.xlarge",
		"tenancy":      "Dedicated",
	}

	lookup := spec.Lookup("us-east-1", attrs)

	// Original attrs preserved
	if lookup.Attributes["instanceType"] != "m5.xlarge" {
		t.Errorf("instanceType overwritten: %q", lookup.Attributes["instanceType"])
	}
	if lookup.Attributes["tenancy"] != "Dedicated" {
		t.Errorf("tenancy overwritten: %q", lookup.Attributes["tenancy"])
	}
	// location added
	if lookup.Attributes["location"] == "" {
		t.Error("location not added")
	}
}
