package awskit

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

func price(rate float64) *pricing.Price {
	return &pricing.Price{OnDemandUSD: rate}
}

func TestCostBuilder_Hourly(t *testing.T) {
	t.Parallel()

	hourly, monthly := NewCostBuilder().Hourly().Calc(price(0.10), nil, "")

	if hourly != 0.10 {
		t.Fatalf("hourly = %v, want 0.10", hourly)
	}
	if monthly != 0.10*costutil.HoursPerMonth {
		t.Fatalf("monthly = %v, want %v", monthly, 0.10*costutil.HoursPerMonth)
	}
}

func TestCostBuilder_HourlyNilPrice(t *testing.T) {
	t.Parallel()

	hourly, monthly := NewCostBuilder().Hourly().Calc(nil, nil, "")

	if hourly != 0 || monthly != 0 {
		t.Fatalf("expected (0, 0), got (%v, %v)", hourly, monthly)
	}
}

func TestCostBuilder_HourlyFallback(t *testing.T) {
	t.Parallel()

	hourly, monthly := NewCostBuilder().Hourly().Fallback(0.045).Calc(nil, nil, "")

	if hourly != 0.045 {
		t.Fatalf("hourly = %v, want 0.045", hourly)
	}
	if monthly != 0.045*costutil.HoursPerMonth {
		t.Fatalf("monthly = %v, want %v", monthly, 0.045*costutil.HoursPerMonth)
	}
}

func TestCostBuilder_HourlyFallbackNotUsedWhenPricePresent(t *testing.T) {
	t.Parallel()

	hourly, _ := NewCostBuilder().Hourly().Fallback(0.045).Calc(price(0.10), nil, "")

	if hourly != 0.10 {
		t.Fatalf("hourly = %v, want 0.10 (price should take precedence)", hourly)
	}
}

func TestCostBuilder_PerUnit(t *testing.T) {
	t.Parallel()

	hourly, monthly := NewCostBuilder().PerUnit(100).Calc(price(0.08), nil, "")

	wantMonthly := 0.08 * 100
	if monthly != wantMonthly {
		t.Fatalf("monthly = %v, want %v", monthly, wantMonthly)
	}
	if hourly != wantMonthly/costutil.HoursPerMonth {
		t.Fatalf("hourly = %v, want %v", hourly, wantMonthly/costutil.HoursPerMonth)
	}
}

func TestCostBuilder_Scale(t *testing.T) {
	t.Parallel()

	hourly, monthly := NewCostBuilder().Hourly().Scale(3).Calc(price(0.10), nil, "")

	if abs(hourly-0.30) > 0.001 {
		t.Fatalf("hourly = %v, want 0.30", hourly)
	}
	if abs(monthly-0.30*costutil.HoursPerMonth) > 0.001 {
		t.Fatalf("monthly = %v, want %v", monthly, 0.30*costutil.HoursPerMonth)
	}
}

func TestCostBuilder_Charge(t *testing.T) {
	t.Parallel()

	hourly, monthly := NewCostBuilder().
		Hourly().Scale(2).
		Charge(NewCharge(100).Fixed(0.10)).
		Calc(price(0.10), nil, "")

	baseMonthly := 0.10 * 2 * costutil.HoursPerMonth
	wantMonthly := baseMonthly + 100*0.10
	wantHourly := wantMonthly / costutil.HoursPerMonth

	if abs(monthly-wantMonthly) > 0.001 {
		t.Fatalf("monthly = %v, want %v", monthly, wantMonthly)
	}
	if abs(hourly-wantHourly) > 0.001 {
		t.Fatalf("hourly = %v, want %v", hourly, wantHourly)
	}
}

func TestCostBuilder_ChargeWithRate(t *testing.T) {
	t.Parallel()

	resolver := func(_ *pricing.PriceIndex, _ string) (float64, bool) {
		return 0.085, true
	}

	_, monthly := NewCostBuilder().
		Hourly().Scale(3).
		Charge(NewCharge(50).Rate(resolver).Fallback(0.10)).
		Calc(price(0.10), &pricing.PriceIndex{}, "us-east-1")

	baseMonthly := 0.10 * 3 * costutil.HoursPerMonth
	wantMonthly := baseMonthly + 50*0.085

	if abs(monthly-wantMonthly) > 0.001 {
		t.Fatalf("monthly = %v, want %v", monthly, wantMonthly)
	}
}

func TestCostBuilder_ChargeZeroQtySkipped(t *testing.T) {
	t.Parallel()

	_, monthly := NewCostBuilder().
		Hourly().
		Charge(NewCharge(0).Fixed(100)).
		Calc(price(0.10), nil, "")

	wantMonthly := 0.10 * costutil.HoursPerMonth
	if abs(monthly-wantMonthly) > 0.001 {
		t.Fatalf("monthly = %v, want %v (zero qty charge should be skipped)", monthly, wantMonthly)
	}
}

func TestCostBuilder_MatchFixed(t *testing.T) {
	t.Parallel()

	hourly, monthly := NewCostBuilder().
		Hourly().
		Match("io1", nil, map[string][]Charge{
			"io1": {NewCharge(1000).Fixed(0.065)},
			"gp3": {NewCharge(3000).Fixed(0.006)},
		}).
		Calc(price(0.10), nil, "")

	baseMonthly := 0.10 * costutil.HoursPerMonth
	wantMonthly := baseMonthly + 1000*0.065
	wantHourly := wantMonthly / costutil.HoursPerMonth

	if abs(monthly-wantMonthly) > 0.001 {
		t.Fatalf("monthly = %v, want %v", monthly, wantMonthly)
	}
	if abs(hourly-wantHourly) > 0.001 {
		t.Fatalf("hourly = %v, want %v", hourly, wantHourly)
	}
}

func TestCostBuilder_MatchUnmatchedKey(t *testing.T) {
	t.Parallel()

	_, monthly := NewCostBuilder().
		Hourly().
		Match("gp2", nil, map[string][]Charge{
			"io1": {NewCharge(1000).Fixed(0.065)},
			"gp3": {NewCharge(3000).Fixed(0.006)},
		}).
		Calc(price(0.10), nil, "")

	wantMonthly := 0.10 * costutil.HoursPerMonth
	if abs(monthly-wantMonthly) > 0.001 {
		t.Fatalf("monthly = %v, want %v (no match should add charges)", monthly, wantMonthly)
	}
}

func TestCostBuilder_MatchFallback(t *testing.T) {
	t.Parallel()

	_, monthly := NewCostBuilder().
		Hourly().
		Match("unknown", []Charge{NewCharge(50).Fixed(0.115)}, map[string][]Charge{
			"io1": {NewCharge(1000).Fixed(0.065)},
		}).
		Calc(price(0.10), nil, "")

	baseMonthly := 0.10 * costutil.HoursPerMonth
	wantMonthly := baseMonthly + 50*0.115

	if abs(monthly-wantMonthly) > 0.001 {
		t.Fatalf("monthly = %v, want %v (should use fallback charges)", monthly, wantMonthly)
	}
}

func TestCostBuilder_MatchFreeTier(t *testing.T) {
	t.Parallel()

	_, monthly := NewCostBuilder().
		PerUnit(100).
		Match("gp3", nil, map[string][]Charge{
			"gp3": {
				NewCharge(5000).FreeTier(3000).Fixed(0.006),
				NewCharge(200).FreeTier(125).Fixed(0.040),
			},
		}).
		Calc(price(0.08), nil, "")

	baseMonthly := 0.08 * 100
	iopsCharge := (5000 - 3000) * 0.006
	tpCharge := (200 - 125) * 0.040
	wantMonthly := baseMonthly + iopsCharge + tpCharge

	if abs(monthly-wantMonthly) > 0.001 {
		t.Fatalf("monthly = %v, want %v", monthly, wantMonthly)
	}
}

func TestCostBuilder_MatchFreeTierSkipsWhenBelowThreshold(t *testing.T) {
	t.Parallel()

	_, monthly := NewCostBuilder().
		PerUnit(100).
		Match("gp3", nil, map[string][]Charge{
			"gp3": {NewCharge(2000).FreeTier(3000).Fixed(0.006)},
		}).
		Calc(price(0.08), nil, "")

	wantMonthly := 0.08 * 100
	if abs(monthly-wantMonthly) > 0.001 {
		t.Fatalf("monthly = %v, want %v (charge should be skipped)", monthly, wantMonthly)
	}
}

func TestCostBuilder_MatchMultipleCharges(t *testing.T) {
	t.Parallel()

	_, monthly := NewCostBuilder().
		Hourly().
		Match("io1", nil, map[string][]Charge{
			"io1": {
				NewCharge(50).Fixed(0.115),
				NewCharge(1000).Fixed(0.10),
			},
		}).
		Calc(price(0.10), nil, "")

	baseMonthly := 0.10 * costutil.HoursPerMonth
	wantMonthly := baseMonthly + 50*0.115 + 1000*0.10

	if abs(monthly-wantMonthly) > 0.001 {
		t.Fatalf("monthly = %v, want %v", monthly, wantMonthly)
	}
}

func TestCostBuilder_MatchWithRateResolver(t *testing.T) {
	t.Parallel()

	resolver := func(_ *pricing.PriceIndex, _ string) (float64, bool) {
		return 0.050, true
	}

	_, monthly := NewCostBuilder().
		PerUnit(100).
		Match("io1", nil, map[string][]Charge{
			"io1": {NewCharge(1000).Rate(resolver).Fallback(0.065)},
		}).
		Calc(price(0.08), &pricing.PriceIndex{}, "us-east-1")

	wantMonthly := 0.08*100 + 1000*0.050
	if abs(monthly-wantMonthly) > 0.001 {
		t.Fatalf("monthly = %v, want %v", monthly, wantMonthly)
	}
}

func TestCostBuilder_MatchWithRateResolverFallback(t *testing.T) {
	t.Parallel()

	resolver := func(_ *pricing.PriceIndex, _ string) (float64, bool) {
		return 0, false
	}

	_, monthly := NewCostBuilder().
		PerUnit(100).
		Match("io1", nil, map[string][]Charge{
			"io1": {NewCharge(1000).Rate(resolver).Fallback(0.065)},
		}).
		Calc(price(0.08), &pricing.PriceIndex{}, "us-east-1")

	wantMonthly := 0.08*100 + 1000*0.065
	if abs(monthly-wantMonthly) > 0.001 {
		t.Fatalf("monthly = %v, want %v (should use rateFallback)", monthly, wantMonthly)
	}
}

func TestCostBuilder_IndexRate(t *testing.T) {
	t.Parallel()

	runtime := NewRuntime(Manifest)
	index := &pricing.PriceIndex{
		Products: map[string]pricing.Price{
			"sku1": {
				ProductFamily: "System Operation",
				Attributes: map[string]string{
					"location":  runtime.ResolveRegionName("us-east-1"),
					"usagetype": "EBS:VolumeP-IOPS.piops",
				},
				OnDemandUSD: 0.065,
			},
		},
	}

	resolver := IndexRate(runtime, "System Operation", "EBS:VolumeP-IOPS.piops")
	rate, ok := resolver(index, "us-east-1")

	if !ok {
		t.Fatal("IndexRate should find the price")
	}
	if rate != 0.065 {
		t.Fatalf("rate = %v, want 0.065", rate)
	}
}

func TestCostBuilder_IndexRateWithRegionalPrefix(t *testing.T) {
	t.Parallel()

	runtime := NewRuntime(Manifest)
	prefix := runtime.ResolveUsagePrefix("eu-west-1")
	index := &pricing.PriceIndex{
		Products: map[string]pricing.Price{
			"sku1": {
				ProductFamily: "System Operation",
				Attributes: map[string]string{
					"location":  runtime.ResolveRegionName("eu-west-1"),
					"usagetype": prefix + "-" + "EBS:VolumeP-IOPS.gp3",
				},
				OnDemandUSD: 0.006,
			},
		},
	}

	resolver := IndexRate(runtime, "System Operation", "EBS:VolumeP-IOPS.gp3")
	rate, ok := resolver(index, "eu-west-1")

	if !ok {
		t.Fatal("IndexRate should find the price with regional prefix")
	}
	if rate != 0.006 {
		t.Fatalf("rate = %v, want 0.006", rate)
	}
}

func TestCostBuilder_IndexRateNilIndex(t *testing.T) {
	t.Parallel()

	runtime := NewRuntime(Manifest)
	resolver := IndexRate(runtime, "System Operation", "EBS:VolumeP-IOPS.piops")
	_, ok := resolver(nil, "us-east-1")

	if ok {
		t.Fatal("IndexRate should return false for nil index")
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
