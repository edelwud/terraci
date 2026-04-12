package storage

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

func TestRoute53Handler_Category(t *testing.T) {
	t.Parallel()

	category := resourcedef.CostCategoryFixed
	handlertest.RunContractSuite(t, resourcespec.MustCompileTyped(Route53Spec()), handlertest.ContractSuite{
		Category:  &category,
		NilLookup: &handlertest.LookupInput{Region: "us-east-1"},
	})
}

func TestRoute53Handler_Describe(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompileTyped(Route53Spec())
	result := def.DescribeResource(nil, nil)
	if result != nil {
		t.Errorf("DescribeResource() = %v, want nil", result)
	}
}

func TestRoute53Handler_BuildLookup_ReturnsNil(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompileTyped(Route53Spec())
	handlertest.AssertNilLookup(t, def, "us-east-1", nil)
}

func TestRoute53Handler_CalculateFixedCost(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompileTyped(Route53Spec())
	hourly, monthly, ok := def.CalculateFixedCost("", nil)
	if !ok {
		t.Fatal("CalculateFixedCost should be available")
	}

	if monthly != Route53HostedZoneCost {
		t.Errorf("monthly = %v, want %v", monthly, Route53HostedZoneCost)
	}

	expectedHourly := Route53HostedZoneCost / costutil.HoursPerMonth
	if hourly != expectedHourly {
		t.Errorf("hourly = %v, want %v", hourly, expectedHourly)
	}
}
