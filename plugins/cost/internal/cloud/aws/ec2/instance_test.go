package ec2

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

func TestInstanceHandler_BuildLookup(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(InstanceSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))

	tests := []struct {
		name        string
		region      string
		attrs       map[string]any
		wantErr     bool
		wantType    string
		wantTenancy string
	}{
		{
			name:   "basic instance",
			region: "us-east-1",
			attrs: map[string]any{
				"instance_type": "t3.micro",
			},
			wantType:    "t3.micro",
			wantTenancy: "Shared",
		},
		{
			name:   "dedicated tenancy",
			region: "eu-central-1",
			attrs: map[string]any{
				"instance_type": "m5.large",
				"tenancy":       "dedicated",
			},
			wantType:    "m5.large",
			wantTenancy: "Dedicated",
		},
		{
			name:    "missing instance_type",
			region:  "us-east-1",
			attrs:   map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.wantErr {
				_, err := def.BuildLookup(tt.region, tt.attrs)
				if err == nil {
					t.Error("BuildLookup should return error")
				}
				return
			}

			lookup := handlertest.RequireLookup(t, def, tt.region, tt.attrs)

			if lookup.Attributes["instanceType"] != tt.wantType {
				t.Errorf("instanceType = %q, want %q", lookup.Attributes["instanceType"], tt.wantType)
			}

			if lookup.Attributes["tenancy"] != tt.wantTenancy {
				t.Errorf("tenancy = %q, want %q", lookup.Attributes["tenancy"], tt.wantTenancy)
			}
		})
	}
}

func TestInstanceHandler_CalculateCost_ComputeOnly(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(InstanceSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))

	price := &pricing.Price{OnDemandUSD: 0.10}

	// CalculateCost now returns compute cost only (no root volume)
	hourly, monthly, ok := def.CalculateStandardCost(price, nil, "", map[string]any{})
	if !ok {
		t.Fatal("CalculateStandardCost should return ok=true")
	}

	expectedMonthly := 0.10 * costutil.HoursPerMonth
	if monthly != expectedMonthly {
		t.Errorf("monthly = %v, want %v (compute only)", monthly, expectedMonthly)
	}
	if hourly != 0.10 {
		t.Errorf("hourly = %v, want %v", hourly, 0.10)
	}
}

func TestInstanceHandler_SubResources_Default(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(InstanceSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))

	// No root_block_device → default 8 GB gp2
	subs := def.BuildSubresources(map[string]any{})

	if len(subs) != 1 {
		t.Fatalf("BuildSubresources() returned %d, want 1", len(subs))
	}
	sub := subs[0]
	if sub.Suffix != "/root_volume" {
		t.Errorf("Suffix = %q, want /root_volume", sub.Suffix)
	}
	if sub.Type != "aws_ebs_volume" {
		t.Errorf("Type = %q, want aws_ebs_volume", sub.Type)
	}
	if costutil.GetStringAttr(sub.Attrs, "type") != awskit.VolumeTypeGP2 {
		t.Errorf("volume type = %q, want %q", costutil.GetStringAttr(sub.Attrs, "type"), awskit.VolumeTypeGP2)
	}
	if costutil.GetFloatAttr(sub.Attrs, "size") != 8 {
		t.Errorf("size = %v, want 8", costutil.GetFloatAttr(sub.Attrs, "size"))
	}
}

func TestInstanceHandler_SubResources_Custom(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(InstanceSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))

	attrs := map[string]any{
		"instance_type": "t3.micro",
		"root_block_device": []any{
			map[string]any{
				"volume_type": "gp3",
				"volume_size": float64(50),
				"iops":        float64(4000),
				"throughput":  float64(200),
			},
		},
	}

	subs := def.BuildSubresources(attrs)

	if len(subs) != 1 {
		t.Fatalf("BuildSubresources() returned %d, want 1", len(subs))
	}
	sub := subs[0]
	if costutil.GetStringAttr(sub.Attrs, "type") != "gp3" {
		t.Errorf("type = %q, want gp3", costutil.GetStringAttr(sub.Attrs, "type"))
	}
	if costutil.GetFloatAttr(sub.Attrs, "size") != 50 {
		t.Errorf("size = %v, want 50", costutil.GetFloatAttr(sub.Attrs, "size"))
	}
	if costutil.GetFloatAttr(sub.Attrs, "iops") != 4000 {
		t.Errorf("iops = %v, want 4000", costutil.GetFloatAttr(sub.Attrs, "iops"))
	}
	if costutil.GetFloatAttr(sub.Attrs, "throughput") != 200 {
		t.Errorf("throughput = %v, want 200", costutil.GetFloatAttr(sub.Attrs, "throughput"))
	}
}

func TestInstanceHandler_Category(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(InstanceSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))
	handlertest.AssertCategory(t, def, resourcedef.CostCategoryStandard)
}

func TestInstanceHandler_Describe(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(InstanceSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))

	tests := []struct {
		name       string
		attrs      map[string]any
		wantKeys   map[string]string
		wantAbsent []string
	}{
		{
			name:       "nil attrs",
			attrs:      nil,
			wantAbsent: []string{"instance_type", "tenancy"},
		},
		{
			name: "instance_type only",
			attrs: map[string]any{
				"instance_type": "t3.micro",
			},
			wantKeys:   map[string]string{"instance_type": "t3.micro"},
			wantAbsent: []string{"tenancy"},
		},
		{
			name: "instance_type and tenancy",
			attrs: map[string]any{
				"instance_type": "m5.large",
				"tenancy":       "dedicated",
			},
			wantKeys: map[string]string{
				"instance_type": "m5.large",
				"tenancy":       "dedicated",
			},
		},
		{
			name: "default tenancy excluded",
			attrs: map[string]any{
				"instance_type": "t3.micro",
				"tenancy":       "default",
			},
			wantKeys:   map[string]string{"instance_type": "t3.micro"},
			wantAbsent: []string{"tenancy"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := def.DescribeResource(nil, tt.attrs)

			for k, v := range tt.wantKeys {
				if result[k] != v {
					t.Errorf("DescribeResource()[%q] = %q, want %q", k, result[k], v)
				}
			}
			for _, k := range tt.wantAbsent {
				if _, ok := result[k]; ok {
					t.Errorf("DescribeResource() should not contain key %q", k)
				}
			}
		})
	}
}

func TestGetRootBlockDevice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		attrs map[string]any
		want  bool
	}{
		{"present", map[string]any{"root_block_device": []any{map[string]any{"volume_size": float64(20)}}}, true},
		{"missing", map[string]any{}, false},
		{"empty list", map[string]any{"root_block_device": []any{}}, false},
		{"wrong type", map[string]any{"root_block_device": "bad"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := getRootBlockDevice(tt.attrs)
			if (got != nil) != tt.want {
				t.Errorf("getRootBlockDevice() returned %v, want non-nil=%v", got, tt.want)
			}
		})
	}
}
