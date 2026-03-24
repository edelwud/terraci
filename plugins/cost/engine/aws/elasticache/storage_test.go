package elasticache

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/engine/pricing"
)

func TestParseGiB(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"13.07 GiB", 13.07},
		{"75 GiB NVMe SSD", 75},
		{"150 GiB NVMe SSD", 150},
		{"0.5 GiB", 0.5},
		{"None", 0},
		{"none", 0},
		{"", 0},
		{"0", 0},
		{"invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseGiB(tt.input)
			if got != tt.want {
				t.Errorf("parseGiB(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNodeMemoryFromPrice(t *testing.T) {
	tests := []struct {
		name  string
		price *pricing.Price
		want  float64
	}{
		{
			name:  "nil price",
			price: nil,
			want:  0,
		},
		{
			name:  "no memory attribute",
			price: &pricing.Price{Attributes: map[string]string{}},
			want:  0,
		},
		{
			name: "valid memory",
			price: &pricing.Price{Attributes: map[string]string{
				"memory": "13.07 GiB",
			}},
			want: 13.07,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nodeMemoryFromPrice(tt.price)
			if got != tt.want {
				t.Errorf("nodeMemoryFromPrice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeSSDFromPrice(t *testing.T) {
	tests := []struct {
		name  string
		price *pricing.Price
		want  float64
	}{
		{
			name: "NVMe SSD",
			price: &pricing.Price{Attributes: map[string]string{
				"storage": "75 GiB NVMe SSD",
			}},
			want: 75,
		},
		{
			name: "no SSD",
			price: &pricing.Price{Attributes: map[string]string{
				"storage": "None",
			}},
			want: 0,
		},
		{
			name:  "nil price",
			price: nil,
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nodeSSDFromPrice(tt.price)
			if got != tt.want {
				t.Errorf("nodeSSDFromPrice() = %v, want %v", got, tt.want)
			}
		})
	}
}
