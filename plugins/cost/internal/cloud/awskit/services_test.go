package awskit

import "testing"

func TestManifest(t *testing.T) {
	if Manifest.ID != ProviderID {
		t.Fatalf("Manifest.ID = %q, want %q", Manifest.ID, ProviderID)
	}
	if Manifest.DisplayName != "Amazon Web Services" {
		t.Fatalf("Manifest.DisplayName = %q, want %q", Manifest.DisplayName, "Amazon Web Services")
	}
	if Manifest.PriceSource != "aws-bulk-api" {
		t.Fatalf("Manifest.PriceSource = %q, want %q", Manifest.PriceSource, "aws-bulk-api")
	}

	service, ok := Service(ServiceKeyEC2)
	if !ok {
		t.Fatal("Manifest.Service(ec2) not found")
	}
	if service != MustService(ServiceKeyEC2) {
		t.Fatalf("Manifest.Service(ec2) = %v, want %v", service, MustService(ServiceKeyEC2))
	}

	if got := Manifest.Regions.ResolveLocationName("eu-west-1"); got != "EU (Ireland)" {
		t.Fatalf("Manifest.Regions.ResolveLocationName(eu-west-1) = %q, want %q", got, "EU (Ireland)")
	}
	if got := Manifest.Regions.ResolveUsagePrefix("eu-west-1"); got != "EUW1" {
		t.Fatalf("Manifest.Regions.ResolveUsagePrefix(eu-west-1) = %q, want %q", got, "EUW1")
	}
	if got := Manifest.Regions.ResolveUsagePrefix("xx-unknown-1"); got != DefaultUsagePrefix {
		t.Fatalf("Manifest.Regions.ResolveUsagePrefix(xx-unknown-1) = %q, want %q", got, DefaultUsagePrefix)
	}
}
