package aws

import "testing"

func TestResolveRegionName_KnownRegions(t *testing.T) {
	tests := []struct {
		region string
		want   string
	}{
		{"us-east-1", "US East (N. Virginia)"},
		{"eu-west-1", "EU (Ireland)"},
		{"eu-central-1", "EU (Frankfurt)"},
		{"ap-southeast-1", "Asia Pacific (Singapore)"},
		{"us-west-2", "US West (Oregon)"},
	}

	for _, tt := range tests {
		got := ResolveRegionName(tt.region)
		if got != tt.want {
			t.Errorf("ResolveRegionName(%q) = %q, want %q", tt.region, got, tt.want)
		}
	}
}

func TestResolveRegionName_UnknownRegion(t *testing.T) {
	got := ResolveRegionName("xx-unknown-1")
	if got != "xx-unknown-1" {
		t.Errorf("ResolveRegionName(unknown) = %q, want %q", got, "xx-unknown-1")
	}
}

func TestResolveRegionName_EmptyString(t *testing.T) {
	got := ResolveRegionName("")
	if got != "" {
		t.Errorf("ResolveRegionName(\"\") = %q, want empty", got)
	}
}

func TestHoursPerMonth(t *testing.T) {
	if HoursPerMonth != 730 {
		t.Errorf("HoursPerMonth = %d, want 730", HoursPerMonth)
	}
}

func TestResolveUsagePrefix_KnownRegions(t *testing.T) {
	tests := []struct {
		region string
		want   string
	}{
		{"us-east-1", "USE1"},
		{"us-east-2", "USE2"},
		{"us-west-1", "USW1"},
		{"us-west-2", "USW2"},
		{"eu-central-1", "EUC1"},
		{"eu-west-1", "EUW1"},
		{"ap-northeast-1", "APN1"},
		{"ap-southeast-1", "APS1"},
		{"sa-east-1", "SAE1"},
		{"ca-central-1", "CAN1"},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			got := ResolveUsagePrefix(tt.region)
			if got != tt.want {
				t.Errorf("ResolveUsagePrefix(%q) = %q, want %q", tt.region, got, tt.want)
			}
		})
	}
}

func TestResolveUsagePrefix_UnknownRegion(t *testing.T) {
	got := ResolveUsagePrefix("xx-unknown-1")
	if got != "USE1" {
		t.Errorf("ResolveUsagePrefix(unknown) = %q, want %q", got, "USE1")
	}
}

func TestResolveUsagePrefix_EmptyString(t *testing.T) {
	got := ResolveUsagePrefix("")
	if got != "USE1" {
		t.Errorf("ResolveUsagePrefix(\"\") = %q, want %q", got, "USE1")
	}
}
