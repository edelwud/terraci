package costutil

import (
	"math"
	"testing"
)

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.001
}

func TestHourlyCost(t *testing.T) {
	h, m := HourlyCost(0.10)
	if !almostEqual(h, 0.10) {
		t.Errorf("hourly = %v, want 0.10", h)
	}
	if !almostEqual(m, 0.10*HoursPerMonth) {
		t.Errorf("monthly = %v, want %v", m, 0.10*HoursPerMonth)
	}
}

func TestHourlyCost_Zero(t *testing.T) {
	h, m := HourlyCost(0)
	if h != 0 || m != 0 {
		t.Errorf("HourlyCost(0) = (%v, %v), want (0, 0)", h, m)
	}
}

func TestScaledHourlyCost(t *testing.T) {
	h, m := ScaledHourlyCost(0.05, 3)
	if !almostEqual(h, 0.15) {
		t.Errorf("hourly = %v, want 0.15", h)
	}
	if !almostEqual(m, 0.15*HoursPerMonth) {
		t.Errorf("monthly = %v, want %v", m, 0.15*HoursPerMonth)
	}
}

func TestScaledHourlyCost_ZeroCount(t *testing.T) {
	h, m := ScaledHourlyCost(0.10, 0)
	if h != 0 || m != 0 {
		t.Errorf("ScaledHourlyCost(0.10, 0) = (%v, %v), want (0, 0)", h, m)
	}
}

func TestScaledHourlyCost_SingleUnit(t *testing.T) {
	h1, m1 := HourlyCost(0.10)
	h2, m2 := ScaledHourlyCost(0.10, 1)
	if h1 != h2 || m1 != m2 {
		t.Errorf("ScaledHourlyCost(x, 1) should equal HourlyCost(x)")
	}
}

func TestFixedMonthlyCost(t *testing.T) {
	h, m := FixedMonthlyCost(73.0)
	if !almostEqual(m, 73.0) {
		t.Errorf("monthly = %v, want 73.0", m)
	}
	if !almostEqual(h, 73.0/HoursPerMonth) {
		t.Errorf("hourly = %v, want %v", h, 73.0/HoursPerMonth)
	}
}

func TestFixedMonthlyCost_Zero(t *testing.T) {
	h, m := FixedMonthlyCost(0)
	if h != 0 || m != 0 {
		t.Errorf("FixedMonthlyCost(0) = (%v, %v), want (0, 0)", h, m)
	}
}

func TestFixedMonthlyCost_Roundtrip(t *testing.T) {
	h, m := FixedMonthlyCost(100.0)
	if !almostEqual(h*HoursPerMonth, m) {
		t.Errorf("roundtrip: %v * %d = %v, want %v", h, HoursPerMonth, h*HoursPerMonth, m)
	}
}
