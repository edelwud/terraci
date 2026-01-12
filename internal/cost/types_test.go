package cost

import "testing"

func TestFormatCost(t *testing.T) {
	tests := []struct {
		cost     float64
		expected string
	}{
		{0, "$0"},
		{0.005, "<$0.01"},
		{0.01, "$0.01"},
		{0.5, "$0.5"},
		{1, "$1"},
		{1.5, "$1.5"},
		{10.25, "$10.25"},
		{100, "$100"},
		{999.99, "$999.99"},
		{1000, "$1,000"},
		{1234.56, "$1,234.56"},
		{10000, "$10,000"},
		{-5.5, "-$5.5"},
		{-0.005, "<$0.01"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatCost(tt.cost)
			if result != tt.expected {
				t.Errorf("FormatCost(%v) = %q, want %q", tt.cost, result, tt.expected)
			}
		})
	}
}

func TestFormatCostDiff(t *testing.T) {
	tests := []struct {
		diff     float64
		expected string
	}{
		{0, "$0"},
		{5.5, "+$5.5"},
		{100, "+$100"},
		{1000, "+$1,000"},
		{-5.5, "-$5.5"},
		{-100, "-$100"},
		{-1000, "-$1,000"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatCostDiff(tt.diff)
			if result != tt.expected {
				t.Errorf("FormatCostDiff(%v) = %q, want %q", tt.diff, result, tt.expected)
			}
		})
	}
}

func TestModuleCost(t *testing.T) {
	mc := ModuleCost{
		ModuleID:   "platform/prod/eu-central-1/vpc",
		ModulePath: "platform/prod/eu-central-1/vpc",
		Region:     "eu-central-1",
		BeforeCost: 100.50,
		AfterCost:  150.75,
		DiffCost:   50.25,
		HasChanges: true,
		Resources: []ResourceCost{
			{
				Address:     "aws_instance.web",
				Type:        "aws_instance",
				MonthlyCost: 50.25,
			},
		},
	}

	if mc.ModuleID != "platform/prod/eu-central-1/vpc" {
		t.Errorf("ModuleID = %q, want %q", mc.ModuleID, "platform/prod/eu-central-1/vpc")
	}

	if mc.ModulePath != "platform/prod/eu-central-1/vpc" {
		t.Errorf("ModulePath = %q, want %q", mc.ModulePath, "platform/prod/eu-central-1/vpc")
	}

	if mc.Region != "eu-central-1" {
		t.Errorf("Region = %q, want %q", mc.Region, "eu-central-1")
	}

	if mc.BeforeCost != 100.50 {
		t.Errorf("BeforeCost = %v, want %v", mc.BeforeCost, 100.50)
	}

	if mc.AfterCost != 150.75 {
		t.Errorf("AfterCost = %v, want %v", mc.AfterCost, 150.75)
	}

	if mc.DiffCost != 50.25 {
		t.Errorf("DiffCost = %v, want %v", mc.DiffCost, 50.25)
	}

	if !mc.HasChanges {
		t.Error("HasChanges should be true")
	}

	if len(mc.Resources) != 1 {
		t.Errorf("len(Resources) = %d, want %d", len(mc.Resources), 1)
	}
}

func TestResourceCost(t *testing.T) {
	rc := ResourceCost{
		Address:     "aws_instance.web",
		Type:        "aws_instance",
		Name:        "web",
		Region:      "us-east-1",
		MonthlyCost: 73.00,
		HourlyCost:  0.10,
		PriceSource: "aws-bulk-api",
		Unsupported: false,
	}

	if rc.Address != "aws_instance.web" {
		t.Errorf("Address = %q, want %q", rc.Address, "aws_instance.web")
	}

	if rc.Type != "aws_instance" {
		t.Errorf("Type = %q, want %q", rc.Type, "aws_instance")
	}

	if rc.Name != "web" {
		t.Errorf("Name = %q, want %q", rc.Name, "web")
	}

	if rc.Region != "us-east-1" {
		t.Errorf("Region = %q, want %q", rc.Region, "us-east-1")
	}

	if rc.MonthlyCost != 73.00 {
		t.Errorf("MonthlyCost = %v, want %v", rc.MonthlyCost, 73.00)
	}

	if rc.HourlyCost != 0.10 {
		t.Errorf("HourlyCost = %v, want %v", rc.HourlyCost, 0.10)
	}

	if rc.PriceSource != "aws-bulk-api" {
		t.Errorf("PriceSource = %q, want %q", rc.PriceSource, "aws-bulk-api")
	}

	if rc.Unsupported {
		t.Error("Unsupported should be false")
	}
}
