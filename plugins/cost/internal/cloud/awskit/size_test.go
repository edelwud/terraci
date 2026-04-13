package awskit

import "testing"

func TestParseGiB(t *testing.T) {
	t.Parallel()

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
			t.Parallel()
			got := ParseGiB(tt.input)
			if got != tt.want {
				t.Errorf("ParseGiB(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
