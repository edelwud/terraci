package aws

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

func TestLookupBuilder_Build(t *testing.T) {
	builder := LookupBuilder{
		Service:       pricing.ServiceEC2,
		ProductFamily: "Compute Instance",
	}

	lookup := builder.Build("us-east-1", map[string]string{
		"instanceType": "t3.micro",
		"tenancy":      "Shared",
	})

	if lookup.ServiceCode != pricing.ServiceEC2 {
		t.Errorf("ServiceCode = %v, want %v", lookup.ServiceCode, pricing.ServiceEC2)
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

func TestLookupBuilder_Build_AddsLocation(t *testing.T) {
	builder := LookupBuilder{
		Service:       pricing.ServiceRDS,
		ProductFamily: "Database Instance",
	}

	lookup := builder.Build("eu-west-1", map[string]string{
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
	expected := ResolveRegionName("eu-west-1")
	if location != expected {
		t.Errorf("location = %q, want %q", location, expected)
	}
}

func TestLookupBuilder_Build_NilAttrs(t *testing.T) {
	builder := LookupBuilder{
		Service:       pricing.ServiceEC2,
		ProductFamily: "NAT Gateway",
	}

	lookup := builder.Build("us-west-2", nil)

	if lookup.Attributes == nil {
		t.Fatal("Attributes should not be nil")
	}
	if lookup.Attributes["location"] == "" {
		t.Error("location should be set even with nil attrs")
	}
}

func TestLookupBuilder_Build_UnknownRegion(t *testing.T) {
	builder := LookupBuilder{
		Service:       pricing.ServiceEC2,
		ProductFamily: "Storage",
	}

	lookup := builder.Build("xx-unknown-1", nil)

	// Unknown region falls back to raw code
	if lookup.Attributes["location"] != "xx-unknown-1" {
		t.Errorf("location = %q, want %q for unknown region", lookup.Attributes["location"], "xx-unknown-1")
	}
	if lookup.Region != "xx-unknown-1" {
		t.Errorf("Region = %q, want %q", lookup.Region, "xx-unknown-1")
	}
}

func TestLookupBuilder_Build_DoesNotOverwriteExistingAttrs(t *testing.T) {
	builder := LookupBuilder{
		Service:       pricing.ServiceEC2,
		ProductFamily: "Compute Instance",
	}

	attrs := map[string]string{
		"instanceType": "m5.xlarge",
		"tenancy":      "Dedicated",
	}

	lookup := builder.Build("us-east-1", attrs)

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
